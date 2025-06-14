package goauth

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/go-oauth2/oauth2/v4"
	"github.com/go-oauth2/oauth2/v4/errors"
	"github.com/go-oauth2/oauth2/v4/models"
	"github.com/kdjuwidja/aishoppercommon/logger"
	"github.com/redis/go-redis/v9"
)

const (
	scriptSHAKey = "SHA:createScript"
)

type JWTTokenStore struct {
	redisClient *redis.Client
	keyCache    map[string]string
	script      string
	hasKeyLimit bool
	maxNumKeys  int
}

func InitializeJWTTokenStore() (oauth2.TokenStore, error) {
	logger.Info("Initializing JWTTokenStore without key limit.")
	return &JWTTokenStore{
		redisClient: nil,
		keyCache:    make(map[string]string),
		script:      "",
		hasKeyLimit: false,
		maxNumKeys:  0,
	}, nil
}

func InitializeJWTTokenStoreWithKeyLimit(redisClient *redis.Client, luaScriptPath string, maxNumKeys int) (oauth2.TokenStore, error) {
	logger.Infof("Initializing JWTTokenStore with key limit = %d.", maxNumKeys)
	// preload the text version of the script, awaiting to be loaded into redis when needed.
	script, err := os.ReadFile(luaScriptPath)
	if err != nil {
		panic(err)
	}

	return &JWTTokenStore{
		redisClient: redisClient,
		keyCache:    nil,
		script:      string(script),
		hasKeyLimit: true,
		maxNumKeys:  maxNumKeys,
	}, nil
}

// Comparison of the number of keys in redis with the maximum number of keys allowed before creating a new token cannot be
// done in a MULTI/EXEC block without race condition. Therefore, the logic is wrapped in a lua script so that the operation
// can be atomic.
func (jwtts *JWTTokenStore) executiveScript(ctx context.Context, script string, keys []string, argv ...interface{}) (string, error) {
	// check if the script is already loaded into redis, if it does, it should have a SHA key stored in redis.
	scriptExists := false
	sha := jwtts.redisClient.Get(ctx, scriptSHAKey).Val()

	scriptSHAExists := sha != ""
	if scriptSHAExists {
		// if the SHA is already stored in redis, check if the script is actually loaded in redis in case redis crashed or restarted at some point.
		sha = jwtts.redisClient.Get(ctx, scriptSHAKey).Val()
		scriptExists = jwtts.redisClient.ScriptExists(ctx, sha).Val()[0]
	}

	if scriptExists {
		// reuses the existing script in redis.
		sha = jwtts.redisClient.Get(ctx, scriptSHAKey).Val()
		reply, err := jwtts.redisClient.EvalSha(ctx, sha, keys, argv...).Result()
		if err != nil {
			return "", err
		}
		return reply.(string), nil
	} else {
		// load the script into redis.
		sha, err := jwtts.redisClient.ScriptLoad(ctx, script).Result()
		if err != nil {
			return "", err
		}
		// put storing the SHA key and executing the script into a MULTI/EXEC block to ensure atomicity.
		tx := jwtts.redisClient.TxPipeline()
		tx.Set(ctx, scriptSHAKey, sha, 0)
		tx.EvalSha(ctx, sha, keys, argv...).Result()
		replies, err := tx.Exec(ctx)
		if err != nil {
			logger.Error("failed to execute script", err.Error())
			return "", err
		}

		reply := replies[1].(*redis.Cmd).Val()
		return reply.(string), nil
	}
}

func (jwtts *JWTTokenStore) Create(ctx context.Context, info oauth2.TokenInfo) error {
	jv, err := json.Marshal(info)
	if err != nil {
		return err
	}

	if jwtts.hasKeyLimit {
		reply, err := jwtts.executiveScript(ctx, jwtts.script, []string{},
			info.GetUserID(),
			strconv.Itoa(jwtts.maxNumKeys),
			info.GetCode(),
			info.GetAccess(),
			info.GetRefresh(),
			fmt.Sprintf("%.0f", info.GetCodeExpiresIn().Seconds()),
			fmt.Sprintf("%.0f", info.GetAccessExpiresIn().Seconds()),
			fmt.Sprintf("%.0f", info.GetRefreshExpiresIn().Seconds()),
			string(jv))
		if err != nil {
			return err
		}

		if reply != "SUCCESS" {
			return errors.New(reply)
		}
	} else {
		jwtts.keyCache["code:"+info.GetCode()] = string(jv)
		jwtts.keyCache["access:"+info.GetAccess()] = string(jv)
		jwtts.keyCache["refresh:"+info.GetRefresh()] = string(jv)
	}

	return nil
}

func (jwtts *JWTTokenStore) getBySearchKeyMatching(ctx context.Context, prefix string, searchKey string) (oauth2.TokenInfo, error) {
	if jwtts.hasKeyLimit {
		keys, err := jwtts.redisClient.Keys(ctx, prefix+":*:"+searchKey).Result()
		if err != nil {
			return nil, err
		}

		if len(keys) == 0 {
			return nil, errors.ErrInvalidAccessToken
		}

		//should only have exactly one key
		key := keys[0]
		data, err := jwtts.redisClient.Get(ctx, key).Result()
		if err != nil {
			return nil, err
		}

		var tokenInfo models.Token
		if err := json.Unmarshal([]byte(data), &tokenInfo); err != nil {
			return nil, err
		}

		return &tokenInfo, nil
	} else {
		key, ok := jwtts.keyCache[prefix+":"+searchKey]
		if !ok {
			return nil, errors.ErrInvalidAccessToken
		}

		var tokenInfo models.Token
		if err := json.Unmarshal([]byte(key), &tokenInfo); err != nil {
			return nil, err
		}

		return &tokenInfo, nil
	}
}

func (jwtts *JWTTokenStore) GetByCode(ctx context.Context, code string) (oauth2.TokenInfo, error) {
	return jwtts.getBySearchKeyMatching(ctx, "code", code)
}

func (jwtts *JWTTokenStore) GetByAccess(ctx context.Context, access string) (oauth2.TokenInfo, error) {
	return jwtts.getBySearchKeyMatching(ctx, "access", access)
}

func (jwtts *JWTTokenStore) GetByRefresh(ctx context.Context, refresh string) (oauth2.TokenInfo, error) {
	return jwtts.getBySearchKeyMatching(ctx, "refresh", refresh)

}

func (jwtts *JWTTokenStore) removeBySearchKeyMatching(ctx context.Context, prefix string, searchKey string) error {
	if jwtts.hasKeyLimit {
		keys, err := jwtts.redisClient.Keys(ctx, prefix+":*:"+searchKey).Result()
		if err != nil {
			return err
		}

		if len(keys) == 0 {
			return errors.ErrInvalidAccessToken
		}

		//should only have exactly one key
		key := keys[0]
		return jwtts.redisClient.Del(ctx, key).Err()
	} else {
		_, ok := jwtts.keyCache[prefix+":"+searchKey]
		if !ok {
			return errors.ErrInvalidAccessToken
		}

		delete(jwtts.keyCache, prefix+":"+searchKey)
		return nil
	}
}

func (jwtts *JWTTokenStore) RemoveByCode(ctx context.Context, code string) error {
	return jwtts.removeBySearchKeyMatching(ctx, "code", code)
}

func (jwtts *JWTTokenStore) RemoveByAccess(ctx context.Context, access string) error {
	return jwtts.removeBySearchKeyMatching(ctx, "access", access)
}

func (jwtts *JWTTokenStore) RemoveByRefresh(ctx context.Context, refresh string) error {
	return jwtts.removeBySearchKeyMatching(ctx, "refresh", refresh)
}
