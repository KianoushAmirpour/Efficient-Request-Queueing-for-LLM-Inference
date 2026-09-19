package infrastructure

import "github.com/redis/go-redis/v9"

var (
	extendLeaseScript = redis.NewScript(`
-- KEYS[1]: sorted set key for processing jobs
-- ARGV[1]: lease extension duration in milliseconds
-- ARGV[2]: job ID

local score = redis.call("ZSCORE", KEYS[1], ARGV[2])
if not score then return 0 end
local t = redis.call("TIME")
local now = tonumber(t[1]) * 1000 + math.floor(tonumber(t[2]) / 1000)
redis.call("ZADD", KEYS[1], "XX", now + tonumber(ARGV[1]), ARGV[2])
return 1
`)

	removeExpiredScript = redis.NewScript(`
-- KEYS[1]: sorted set key for processing jobs
-- ARGV[1]: job ID
-- ARGV[2]: cutoff timestamp in milliseconds

local score = redis.call("ZSCORE", KEYS[1], ARGV[1])
if not score or tonumber(score) > tonumber(ARGV[2]) then return 0 end
return redis.call("ZREM", KEYS[1], ARGV[1])
`)

	requeueExpiredScript = redis.NewScript(`
-- KEYS[1]: sorted set key for processing jobs 
-- KEYS[2]: sorted set key for active users
-- KEYS[3]: list key for user-specific queue 
-- KEYS[4] = rotation sequence key
-- ARGV[1]: job ID
-- ARGV[2]: user ID
-- ARGV[3]: cutoff timestamp in milliseconds

-- 1. Verify the job is expired and still in the processing set
local score = redis.call("ZSCORE", KEYS[1], ARGV[1])
if not score or tonumber(score) > tonumber(ARGV[3]) then 
    return 0 
end

-- 2. Remove from processing and add to the user's queue
redis.call("ZREM", KEYS[1], ARGV[1])
redis.call("LPUSH", KEYS[3], ARGV[1])

-- 3. If the queue was previously empty, activate the user
if redis.call("LLEN", KEYS[3]) == 1 then
    local rotation = redis.call("INCR", KEYS[4])
    redis.call("ZADD", KEYS[2], "NX", rotation, ARGV[2])
end

return 1
`)

	fairDequeueScript = redis.NewScript(`
-- KEYS[1]: sorted set key for active users
-- KEYS[2]: sorted set key for processing jobs
-- KEYS[3]: rotation sequence key
-- ARGV[1]: lease duration in milliseconds

if redis.call("ZCARD", KEYS[1]) == 0 then
    return nil
end

local user = redis.call("ZRANGE", KEYS[1], 0, 0)[1]
-- per-user queue is dynamically constructed
local job = redis.call("RPOP", "queue:user:" .. user)

if not job then
    redis.call("ZREM", KEYS[1], user)
    return nil
end

local remaining = redis.call("LLEN", "queue:user:" .. user)

if remaining > 0 then
    -- More jobs: update score to move user to end of round-robin
    local score = redis.call("INCR", KEYS[3])
    redis.call("ZADD", KEYS[1], score, user)
else
    -- No more jobs: remove user from active set entirely
    redis.call("ZREM", KEYS[1], user)
end
-- Track job in single sorted set using a lease deadline.
local t = redis.call("TIME")
local now = tonumber(t[1]) * 1000 + math.floor(tonumber(t[2]) / 1000)
local lease_deadline = now + tonumber(ARGV[1])
redis.call("ZADD", KEYS[2], lease_deadline, job)

return {user, job}
	`)

	enqueueScript = redis.NewScript(`
-- KEYS[1] = idempotency key
-- KEYS[2] = user queue key
-- KEYS[3] = active user set
-- KEYS[4] = rotation sequence key

-- ARGV[1] = job id
-- ARGV[2] = idempotency ttl
-- ARGV[3] = max queue length
-- ARGV[4] = userID

local queueLength = redis.call("LLEN", KEYS[2])

if queueLength >= tonumber(ARGV[3]) then
    return { 0 } -- queue full
end

local created = redis.call("SET", KEYS[1], "1", "NX", "EX", tonumber(ARGV[2]))

if not created then
    return { 1 } -- duplicate job
end

redis.call("LPUSH", KEYS[2], ARGV[1])

if queueLength == 0 then

    local score = redis.call("INCR", KEYS[4])

    redis.call("ZADD", KEYS[3], "NX", score, ARGV[4])

    return { 2 } -- job became active
end

return { 3 } -- job enqueued successfully but not active
`)

	enqueueOrphanScript = redis.NewScript(`
-- KEYS[1]: completed jobs set (Set)
-- KEYS[2]: processing jobs set (Sorted Set)
-- KEYS[3]: user-specific queue (List)
-- KEYS[4]: active users set (Sorted Set)
-- KEYS[5]: rotation sequence key
-- ARGV[1]: job ID
-- ARGV[2]: user ID
-- ARGV[3]: max queue capacity per user

if redis.call("SISMEMBER", KEYS[1], ARGV[1]) == 1 then return 0 end -- Already completed
if redis.call("ZSCORE", KEYS[2], ARGV[1]) then return 0 end       -- Currently processing
if redis.call("LPOS", KEYS[3], ARGV[1]) then return 0 end         -- Already in queue

if redis.call("LLEN", KEYS[3]) >= tonumber(ARGV[3]) then return 2 end -- Queue full

local wasEmpty = redis.call("LLEN", KEYS[3]) == 0
redis.call("LPUSH", KEYS[3], ARGV[1])

if wasEmpty then
    local seq = redis.call("INCR", KEYS[5])
    redis.call("ZADD", KEYS[4], "NX", seq, ARGV[2])
end

return 1
`)
)
