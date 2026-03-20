-- Channel concurrency acquire script
-- KEYS[1]: channel concurrency key
-- ARGV[1]: max concurrency limit
-- Returns: 1 if acquired, 0 if rejected

local key = KEYS[1]
local maxConcurrency = tonumber(ARGV[1])

local current = tonumber(redis.call('GET', key) or '0')

if current < maxConcurrency then
    redis.call('INCR', key)
    redis.call('EXPIRE', key, 300) -- 5 min TTL as safety net
    return 1
end

return 0
