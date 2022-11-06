# JRpcFrame
JRpcFrame是一款用go语言实现的rpc框架。

JRpcFrame特点：

- 总体设计如go语言设计一样，尽可能的提供简洁和易用的模式。
- 将整个框架抽象两大对象，node,service。通过统一的组合模型管理各功能模块的关系。
- 能将不同的service配置到不同的node，并能高效的协同工作。
- service的处理通过event驱动，不同的event不同的处理方式，并会对每个event监测，防止异常event。
- 实现了rpc的基本功能，包括rawgo，go，call，asyncall等方法，同时实现了动态服务发现，实时监测各个节点的状态。
- 提供了一些基础模块，包括日志，时间轮定时器和配置模块。




