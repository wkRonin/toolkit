--1. 检查是不是自己的锁
--2. 删除
-- KEYS[1]:分布式锁的key
-- ARGV[1]：预期的存在redis中的value
if redis.call('get', KEYS[1]) == ARGV[1]
then
-- 是自己的锁
    return redis.call('del', KEYS[1])
else
-- 不是自己的锁
    return 0
end