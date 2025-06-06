package defaults

import "github.com/kdjuwidja/aishoppercommon/osutil"

var DEFAULT_API_CLIENTS = []map[string]interface{}{
	{
		"id":          "82ce1a881b304775ad288e57e41387f3",
		"secret":      "my_secret",
		"domain":      "http://localhost:3000",
		"is_public":   true,
		"description": "Default client for ai_shopper_depot",
		"scopes":      "profile shoplist search",
	},
	{
		"id":          "de0125bfee1a486385819cdbb95ac675",
		"secret":      "1ecc882ca7c24701bf7f201f366b5fe8",
		"domain":      "http://localhost:3000",
		"is_public":   true,
		"description": "Default admin client for ai_shopper_depot",
		"scopes":      "admin",
	},
}

var DEFAULT_ROLES = []map[string]interface{}{
	{
		"id":          1,
		"description": "admin",
		"scopes":      []string{"admin"},
	},
	{
		"id":          2,
		"description": "regular users",
		"scopes":      []string{"profile", "shoplist", "search"},
	},
}

var DEFAULT_USERS = []map[string]interface{}{
	{
		"id":       "eb5dc850f1fb40a8b9b2bffd89c6a32d",
		"email":    osutil.GetEnvString("DEFAULT_USER_1_EMAIL", ""),
		"password": osutil.GetEnvString("DEFAULT_USER_1_PASSWORD", ""),
		"roles":    []int{1, 2},
	},
	{
		"id":       "73064f370eda46a48a86e1fd8118be4c",
		"email":    osutil.GetEnvString("DEFAULT_USER_2_EMAIL", ""),
		"password": osutil.GetEnvString("DEFAULT_USER_2_PASSWORD", ""),
		"roles":    []int{1, 2},
	},
}
