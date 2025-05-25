# 分布式缓存
## 前言
在本项目中，很多部分参考了[geketutu](https://geektutu.com/post/geecache.html)的实现和`groupcache`的实现。与`tutu`的`geecache`相比，本项目实现了一些全新的功能，分别包括：
- LRU
- ttl
- grpc
- expired cache eviction
后续还会实现`etcd`服务注册(~~这里留一个坑~~), 填坑！
本项目主要作为一个教学或者基础项目，所以没有方便的客户端来供使用，本人将从学习的角度来讲解一下项目的思路以及实现流程。

## LRU与cache eviction
对于缓存来说，其空间不是无限大的，当缓存的值达到一定的阈值后，就要发生缓存驱逐。
常见的驱逐算法有`FIFO(First In First Out)、LRU(Least RecentLy Used)和LFU(Least Frequently Used)`。但是，由于缓存是“金子一样珍贵”的东西，如果被驱逐的数据是热点数据(即多个请求可能请求的值)，那么当再次发起访问的时候，就会发生`cache miss`，从而可能增加数据库的压力，有可能导致服务宕机！所以一个合理且高效的缓存驱逐策略是很必要的，下面来分析一下常见的三种算法。
### FIFO
`FIFO`算法也就是队列的思想，先缓存的值在驱逐的时候优先被驱逐。这个算法的实现思路很简单，只需要根据先后顺序来维护一个队列即可。
![FIFO](https://raw.githubusercontent.com/XiaoPeng0x3/blogImage/main/20250520223216.png)
假设此时缓存达到上限，要发生缓存驱逐，那么，根据这个原则，优先到来的缓存`key1`就要被驱逐，此时就可以添加一个新的缓存。
![](https://raw.githubusercontent.com/XiaoPeng0x3/blogImage/main/20250520223713.png)
显然，`FIFO`的缺点就是只是根据先后顺序去进行删除，如果`key1`是一个热点数据，那么这个`cache`就会发生`miss`。所以，我们不采用这种算法。

### LRU
`LRU`的数据结构一般来说是`Map + DoubleLinkList`，`Map`可以提供`O(1)`查询， 而`DoubleLinkList`可以将热点`key`放到队首来，这样可以保证队首的数据总是热点数据，这样在`key`驱逐的时候，只需要淘汰队尾数据即可。

![](https://raw.githubusercontent.com/XiaoPeng0x3/blogImage/main/20250520230643.png)

### LFU
LFU在wiki上的描述就是维护一个内部计数器，当`cache hit`的时候，就把这个引用计数+1，这样，当达到容量后，引用计数最小的那个`cache`就该被驱逐。
>Wiki: The simplest method to employ an LFU algorithm is to assign a counter to every block that is loaded into the cache. Each time a reference is made to that block the counter is increased by one. When the cache reaches capacity and has a new block waiting to be inserted the system will search for the block with the lowest counter and remove it from the cache, in case of a tie (i.e., two or more keys with the same frequency), the Least Recently Used key would be invalidated.

这样的思路很直观，但是请考虑一种情况，当一个`key`块在某一段时间内被一直访问，之后却不再访问。也就是说，这个`key`块有着一个次数很多的引用计数，即使它之后没有被访问；当发生`key`驱逐的时候，这个`key`块也不会被驱逐，但是在这一段时间内，这个块并不是热点数据，也就是说，`LFU`并不可以很好的反映**热点**数据。

所以，我们折中一下，选择比较完美的`LRU`

### LRU-K
`LRU-K`这个`k`的含义就是指访问k次之后，这个`key`才可以变为热点数据，从而实现热点数据追踪。其思路就是维护两个`cache`块，一个是`history cache`，另一个就是`hot cache`。当请求到来时，会先到达`history cache`，当这个请求的次数大于`k`次后，就会移动到`hot cache`中，因此，可以将访问次数封装为一个`cache`字段。

```go
// Real data that stored in cache
type entry struct {
	key   string
	visit int
	value Value
}
```

## 自动清理过期缓存
对于一个`cache`来说，自动清理过期数据是很有必要的，清理过期数据可以节省大量的空间，从而更少的发生`cache eviction`，不过，自动清理的时间也不宜设置的过短，否则也会发生`cache miss`。

在go语言中，可以轻松使用多线程，当我们启动一个`cache`服务后，就可以使用多线程来清理缓存。
### 过期策略
过期策略的思路就是去记录`key`的过期时间，每次在`Get`缓存的时候，都要对`key`进行是否过期的判断。

### 实现定时策略
定时清理也就是一个定时器触发的一个任务，创建定时器可以通过`time.NewTicker`来创建，这样，当定时器触发时，就可以把“触发的信号”发送到定时器的`chan`中来作为触发定时器的标志。
```Go
func (c *cache) startEvictionLoop(interval time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.evictionRunning {
		return
	}
	c.evictionRunning = true
	c.stopChan = make(chan struct{})
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				c.mu.Lock()
				if c.lru != nil {
					c.lru.CleanExpired()
				}
				c.mu.Unlock()
			case <-c.stopChan:
				return
			}
		}
	}()
}
```

## 分布式结点设计
### 存在的问题
分布式系统需要特别注意的一点就是数据的获取。假设这样一种情形，这里总共有三台机器`A`、`B`、`C`，在每台机器上都运行了缓存服务。当并发请求时，如果不作任何限制，那么假设并发请求的都是同一个`key`，此时并不能确定这个`key`究竟是去请求哪一台机器？所以，当请求被随机转发后，`A`、`B`、`C`三台机器都可能保存同一份`key`的缓存，造成数据冗余。

而且，由于转发是随机的，当一个请求到来时，例如`key1`缓存存在于`A`机器种，但是这个请求却转发给了`B`服务，导致`Cache miss`。

所以，我们要思考一下这两个问题：
- 数据冗余
- 同一个`key`只会由同一个机器处理

### hash算法
固定结点位置的操作很容易联想到`hash`算法，我们假设一个`hash`算法是`hash(key) % n`，其中，`n`指服务器的台数。对于上面这个问题，`n`的数量和`hash(key)`的值都是固定的，所以每次请求也会将请求发送给固定的机器，似乎解决了问题！
![](https://raw.githubusercontent.com/XiaoPeng0x3/blogImage/main/20250521200702.png)
但是结点的宕机并不是人为可以预测的，假设“A"服务宕机，那么原本属于`A`结点的所有数据不得不去重新计算`hash`值，即`hash(key) % 2`。这样就会导致基本上所有数据都要重复计算`hash`值，导致数据大量迁移。



对于用户来说，结点如何分配是不感知的，但是他们就会发现似乎发现响应时间变长了！差评！
### 一致性hash
一致性哈希就是解决这个问题的！一致性`hash`的算法思路是：维护一个 `0 ~ 2^32-1`范围的一个`hash`值，对于请求`key`来说，首先计算其`hash`值，然后把这个查询`key`的任务分配给**顺时针遇到的第一个结点**。
![](https://raw.githubusercontent.com/XiaoPeng0x3/blogImage/main/20250521201754.png)

如图所示"key1, key2"会被分向"A"结点， "key3, key4, key5"会被分往"B"种，"key6"则会被分向"C"中，这样即使某个服务宕机，那也是某一个区间内的值发生迁移！

### 负载均衡
可能你已经想到了，即使这样做也会导致一些问题，假设我们的`hash`算法很烂，也就是说`A, B, C`这三台机器在`hash`环上的分布不够均匀分散，那么也会导致数据大量迁移的问题。
![](https://raw.githubusercontent.com/XiaoPeng0x3/blogImage/main/20250521202358.png)

如上图，"A"服务宕机后，大量结点顺时针找到的第一个服务是"B"服务，导致数据大量迁移。

一个合理的办法是给结点添加虚拟结点，我们可以维护一个虚拟地址到真实结点之间的一个`hash`表，这样当请求到来时，就可以达到负载均衡的效果。
![](https://raw.githubusercontent.com/XiaoPeng0x3/blogImage/main/20250521203541.png)

可以看到，即使某个结点宕机，也只会造成一小部分数据需要重新移动！

## 分组设计
分组设计的一个好处就是分流。对于不同的请求，例如`scores, name`等请求，如果没有分组设计，那么这些请求都会去请求总的机器。
![](https://raw.githubusercontent.com/XiaoPeng0x3/blogImage/main/20250521231105.png)

在分组之后，不同的`group`维护一个各自的`cache`，各个请求之间互补打扰。

![](https://raw.githubusercontent.com/XiaoPeng0x3/blogImage/main/20250521231512.png)

其实`group`可以理解为`redis`里面的一个`db`，各个`group`之间需要实现通信功能。

其实这也是分布式结点最重要的一部分，其调用流程是
> 先从本地缓存开始查找，如果`cache hit`，那么直接返回数据；否则，就去调用其它在线的结点，从其它结点获取。

而结点之间通信的方式我们可以使用`rpc`。

## 解决缓存击穿
在缓存中，永远有三个绕不开的话题，即`缓存击穿、缓存穿透、缓存雪崩`

> 缓存击穿指的是热点`key`过期的时候出现的情况。当很多请求并发请求一个`key`时，就可以把这个`key`当作热点数据，当这个`key`过期后，由于`cache miss`，请求就会发送到数据库，而数据库往往承受不住大量的请求，造成服务宕机。

> 缓存穿透指的是请求缓存和数据库中都不存在的数据。当请求一个不存在的`key`时，请求最后由数据库返回给缓存(空值,因为key不存在)，导致数据库宕机。

> 缓存雪崩指的是过期时间的设置造成的数据库宕机。当缓存中的大量数据在某一时刻同时过期，那么这些数据都要去数据库中进行查询，造成数据库宕机。

对于这三种情况，我们来分别讨论一下应对策略。
- 缓存击穿

	虽然是很多请求，但是实际上大家请求的东西都是同一个`key`，也就是说，只需要让第一个请求去请求即可，其它请求等待返回或者返回一个旧值即可，也就是**互斥锁/singleflight**方案。

- 缓存穿透

	缓存穿透更像是别有用心的攻击。这类解决方法很多，例如可以对请求值加以判断，或者给数据库设置空值(`NullValue`)，直接返回空值(**可能会污染数据库！**)。还可以使用布隆过滤器来判断请求值是否存在于数据库或缓存中。

- 缓存雪崩

	缓存雪崩是大量数据过期所导致的，所以，可以将过期时间错开，或者考虑将热点数据设置为永不过期，还可以设置多级缓存，从而减少数据库压力。

## 服务注册与服务发现
使用`etcd`作为服务发现和服务注册。在启动项目时，可以将在线结点信息写入`etcd`里面，当进行结点选择时时，通过客户端与`etcd`建立连接，从`etcd`中拿到需要的服务地址，然后返回客户端。

# 总结
这个项目比较重要的部分本人已经讲解完毕，更多细节都藏在代码中，欢迎大家讨论！