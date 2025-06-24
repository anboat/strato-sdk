![StratoSDK-Light](assets/logo.png)

[English](README.md) | 中文

**StratoSDK** 是一款开箱即用的深度研究解决方案，旨在解决当前软件开发中缺乏统一研究工具的问题。通过提供标准化的 SDK 和接口规范，它可以帮助开发人员快速集成深度研究能力，而无需重复造轮子。

## 特性
在轻量版中，我们专注于网络搜索，并提供以下功能：
- 智能网页解析：支持 Jina AI、Firecrawl 等多种内容处理引擎
- 深度问题分析：自动进行问题反思和查询拆分，以提高答案质量
- 多平台搜索：内置对 Twitter 等社交平台的搜索功能
- 灵活配置：可自定义搜索深度和广度设置

## 支持的搜索平台
| 平台               | 类型   |    状态   |
| ---------------------- |------| ----------- |
| [Firecrawl](https://www.firecrawl.dev)               | 网页搜索 API |    :white_check_mark:   |
| [Jina](https://jina.ai)               | 网页搜索 API |    :white_check_mark:   |
| [Twitter](https://x.com/home)               | 社交媒体 |    :white_check_mark:   |
| [Searxng](https://github.com/searxng/searxng)               | 自托管搜索引擎 |    :white_check_mark:   |
| RedNote               | 社交媒体 |    :construction:   |
| Bing               | 搜索引擎 |    :construction:   |
| Google               | 搜索引擎 |    :construction:   |


## 快速开始
该项目主要是为了方便快捷地实现深度搜索功能，并提供了高度的配置定制化。用户可以基于该项目扩展开发自己的第三方搜索工具和网页爬取工具。本项目目前处于第一个版本开发阶段，代码的许多细节处理得非常粗糙，未来会逐步完善。

### 安装
```go
go get github.com/anboat/strato-sdk@latest
```
### 示例
#### 导入
```go
package main
import (
    "context"
    "fmt"
    "os"
    "os/signal"
    "github.com/anboat/strato-sdk/config"
    "github.com/anboat/strato-sdk/core/agent"
    "github.com/anboat/strato-sdk/pkg/logging"
    "syscall"
    "time"
)
```
#### 使用案例
```go
// 加载配置文件
allConfig := config.LoadConfig("config.yaml")
if allConfig == nil {
    fmt.Printf("加载配置失败\n")
    return
}
// 根据加载的配置初始化日志记录器
logging.InitLoggerFromConfig(&allConfig.Log)
// 创建一个后台上下文
ctx := context.Background()
// 创建一个新的流式研究代理
rAgent, err := agent.NewStreamingResearchAgent(ctx)
if err != nil {
    fmt.Printf("创建流式研究代理失败: %v\n", err)
    return
}
// 定义研究查询
query := "2024年人工智能的未来是什么？"

// 执行流式研究过程
thoughtChan, err := rAgent.ResearchWithStreaming(ctx, query)

```
## 项目结构
```python
strato-sdk/
├── adapters/         # 各种适配器（搜索、大语言模型、网页抓取等）
│   ├── llm/          # 大语言模型适配器
│   ├── search/       # 搜索引擎适配器（例如 SearxNG、Firecrawl、Twitter 等）
│   └── web/          # 网页抓取适配器（例如 Jina、Firecrawl 等）
├── config/           # 配置相关（YAML 配置、加载器、类型定义）
├── core/             # 核心业务逻辑
│   └── agent/        # 智能代理相关（流式研究代理、工具等）
│   └── tool/         # 代理中调用的工具，例如用于爬取网页内容的 web_process_tool 和用于搜索的 search_tool
├── pkg/              # 通用工具包（例如日志记录）
├── example_main.go   # 示例主程序
├── go.mod            # Go 模块定义
├── README.md         # 项目文档
```

## 技术介绍
### Golang
当前项目完全使用 Go 语言开发，所有核心逻辑、适配器、配置、工具等均使用 Go 实现。
### 第三方依赖
本项目主要使用的第三方依赖有：eino、viper、zap。
- **Eino**：Cloudwego 的 Eino 框架主要用于流程编排，处理各个流程节点之间的流转和跳转过程，例如生成子问题，流转到搜索节点，再获取网页内容，并使用模型判断当前答案的质量以选择跳转。
- **Viper**：主要用于读取配置文件和管理配置。
- **Zap**：主要用于日志管理和实现。
### 设计模式
![Design_Arch](assets/design_arch.png)
以搜索模块为例，简单说明一下代码的设计理念。搜索模块使用了适配器模式、工厂模式、策略模式和注册表模式。所有引擎都实现了具有统一接口的 SearchAdapter 接口，以屏蔽不同搜索引擎之间的差异，实现标准化。工厂模式可以根据配置文件自动注册和创建适配器。注册表模式提供了一个全局注册表来管理所有适配器。不同的搜索引擎可以作为插件动态注册（需要在代码中引入自定义实现的搜索引擎包来调用 init 方法以自动注册到注册表中）。您也可以手动调用相应的方法进行手动注册。策略模式提供了不同的搜索策略，例如混合搜索、降级等。

## 许可证
该仓库遵循 [MIT 许可证](LICENSE)。

## 联系我们
我们是 **Anboat**，一个致力于推动智能体应用开发前沿的初创团队。作为 AI 生态系统的热情建设者，我们专注于创建能够弥合复杂 AI 能力与实际业务应用之间差距的智能解决方案。
我们的目标是通过为开发人员提供强大、易于使用的工具来加速智能应用的开发，从而实现 AI 智能体技术的民主化。我们相信，每个开发人员都应该有能力利用 AI 智能体的力量，而不会迷失在技术复杂性中。
- 邮箱: coder@anboat.cn
- GitHub: https://github.com/anboat
- Twitter: @Anboat240326

## 致谢
我们衷心感谢 Eino 团队 https://github.com/cloudwego/eino 对 AI 智能体开发生态系统的杰出贡献。 
