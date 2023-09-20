# game tools

这个仓库用于提供一些常用的游戏开发中需要的代码生成工具。

考虑到go开发中常常需要使用interface, reflect来实现一些功能，但是效率比较低。
开发者也可以通过手写代码的方式来实现同样的逻辑，但是开发效率比较低下，且不方便统一
维护，因此本仓库实现一些常用逻辑的生成代码，来辅助开发。

本库包含的commands:

- [`cmd/event_trigger`](https://github.com/LeGamerDc/game_tools/tree/main/cmd/event_trigger) 用于生成`EventTable`用于提供基于事件的开发逻辑。
- [`gd`](https://github.com/LeGamerDc/game_tools/tree/main/gd) 用于方便读取配置文件，支持更新、懒加载、高性能。