-- Variables
local userID = ARGV[1]
local maxNumKeys = tonumber(ARGV[2])
local code = ARGV[3]
local access = ARGV[4]
local refresh = ARGV[5]
local codeTTL = ARGV[6]
local accessTokenTTL = ARGV[7]
local refreshTokenTTL = ARGV[8]
local tokenInfo = ARGV[9]

-- Get all keys matching the access token pattern for a user
local reply = redis.pcall('KEYS', "access:" .. userID .. ":*")
if reply['err'] ~= nil then
    redis.log(redis.LOG_WARNING, reply['err'])
    return 'ERROR: failed to get access tokens'
end

local keys = 0
if reply['res'] ~= nil then
    keys = #reply['res']
    if keys >= maxNumKeys then
        return "ERROR: too many access tokens"
    end
end


local keyId = keys + 1
local err = false

if code ~= nil and code ~= '' then
    local codeKey = "code:" .. userID .. ":" .. keyId .. ":" .. code
    local codeReply = redis.pcall('SET', codeKey, tokenInfo, "EX", codeTTL)
    redis.log(redis.LOG_DEBUG, "SETTING CODE KEY: " .. codeKey .. " " .. tokenInfo .. " " .. codeTTL)
    if codeReply['err'] ~= nil then
        redis.log(redis.LOG_WARNING, "CODE ERROR: " ..codeReply['err'])
        err = true
    end
end

if access ~= nil and access ~= '' then
    local accessKey = "access:" .. userID .. ":" .. keyId .. ":" .. access
    local accessReply = redis.pcall('SET', accessKey, tokenInfo, "EX", accessTokenTTL)
    redis.log(redis.LOG_DEBUG, "SETTING ACCESS KEY: " .. accessKey .. " " .. tokenInfo .. " " .. accessTokenTTL)
    if accessReply['err'] ~= nil then
        redis.log(redis.LOG_WARNING, "ACCESS ERROR: " ..accessReply['err'])
        err = true
    end
end

if refresh ~= nil and refresh ~= '' then
    local refreshKey = "refresh:" .. userID .. ":" .. keyId .. ":" .. refresh
    local refreshReply = redis.pcall('SET', refreshKey, tokenInfo, "EX", refreshTokenTTL)
    redis.log(redis.LOG_DEBUG, "SETTING REFRESH KEY: " .. refreshKey .. " " .. tokenInfo .. " " .. refreshTokenTTL)
    if refreshReply['err'] ~= nil then
        redis.log(redis.LOG_WARNING, "REFRESH ERROR: " ..refreshReply['err'])
        err = true
    end
end

if err then
    return "ERROR: failed to set token info"
end

return "SUCCESS"