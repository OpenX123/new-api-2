-- Channel concurrency release script
-- KEYS[1]: channel concurrency key
-- Returns: current count after release

local key = KEYS[1]
local current = tonumber(redis.call('GET', key) or '0')

if current > 0 then
    return redis.call('DECR', key)
end

return 0
