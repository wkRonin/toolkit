# toolkit
1. 泛型工具包
2. 各种框架的拓展插件、callback、可观测性等

## containerx
描述：拓展的数据容器
1. 缩容机制
2. 小顶堆实现的优先队列
3. TODO: 并发安全的优先队列

## grpcx
描述：grpc的拓展
1. server端快捷启动封装
2. etcd作为注册中心的注册封装
3. 实现consul的resolver
4. consul作为注册中心的注册封装
5. 基于grpc的接口实现自己的权重负载均衡算法
6. grpc的截断器
   - 使用本库的logger实现服务端和客户端的日志记录
   - 客户端和微服务之间的链路追踪
   - 客户端和服务端的metric指标采集

## ginx
描述：：gin中间件、统一处理error日志
1. 日志中间件
   - 记录请求方法、请求路径、请求体、响应体、耗时、响应码
2. 带日志的recovery中间件
3. 限流中间件
   - 使用本库ratelimit的方法封装成gin的中间件
4. prometheus埋点
   - 采集当前活跃请求数
   - 采集http接口响应时间
   - 错误码统计(在第5条的统一处理中埋点)
5. 统一处理请求体bind/错误日志打印/ctx中取值

## gormx
1. 使用gorm的callback 采集增删改查的sql响应时间提供给prometheus采集
2. 利用callback实现的gorm plugin ，用来替换gorm自带的logger

## ratelimit
1. 使用滑动窗口算法的lua脚本实现限流接口

## redisx
1. 实现redis的hook接口：prometheus埋点redis命令的响应时间
2. 实现redis的hook接口：opentelemetry 的 trace 埋点
3. 使用redis实现分布式锁

## saramax
实现sarama的ConsumerGroupHandler接口
1. 单个消费就提交：ConsumeClaim中封装解析消息体、记录日志、提交消费
2. 批量消费提交：ConsumeClaim中封装解析消息体、记录日志、提交消费

## syncx
1. 泛型封装atomic.Value（load方法性能比原生的差1倍，其他方法只差一点点可忽略不计）

## zapx
1. 封装uber的zap库


