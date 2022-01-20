package redisOperation

import "gopkg.in/redis.v5"

//
// Del
//  @Description: redis删除数据
//  @param client
//  @param key
//  @return *redis.IntCmd
//
func Del(client *redis.Client, key string) *redis.IntCmd {
    intCmd := client.Del(key)
    return intCmd
}

//
// Get
//  @Description: redis获取数据
//  @param client
//  @param key
//  @return *redis.StringCmd
//
func Get(client *redis.Client, key string) *redis.StringCmd {
    cmd := redis.NewStringCmd("GET", key)
    client.Process(cmd)
    return cmd
}

//
// Set
//  @Description: redis存储数据
//  @param client
//  @param key
//  @param value
//  @return *redis.StringCmd
//
func Set(client *redis.Client, key string, value string) *redis.StringCmd {
    cmd := redis.NewStringCmd("SET", key, value)
    client.Process(cmd)
    return cmd
}

//
// Keys
//  @Description: redis寻找数据
//  @param client
//  @param key
//  @return *redis.StringSliceCmd
//
func Keys(client *redis.Client, key string) *redis.StringSliceCmd {
    stringSliceCmd := client.Keys(key+":*")
    return stringSliceCmd
}

//
// SetEX
//  @Description: redis存储数据
//  @param client
//  @param key
//  @param value
//  @return *redis.StringCmd
//
func SetEX(client *redis.Client, key string, time string, value string) *redis.StringCmd {
    cmd := redis.NewStringCmd("SET", key, time, value)
    client.Process(cmd)
    return cmd
}
