## gd config

`gd` 用于拉取配置文件并监听配置文件更新及时刷新配置文件，具有`懒加载`，`依赖处理`，`自动更新`的特点：

1. 懒加载，用户配置多个配置文件及解析代码，但直到用户显示调用GetDoc时才会首次加载到内存，对于大型项目的各个微服务可以节省内存
不存储不必要的配置。
2. 当文件发生更新，`gd`会自动重置配置文件内存，下次读取的时会引发重载。
3. 依赖处理，一些配置文件依赖于另外一些配置文件的内容，当被依赖的文件更新后，会按拓扑排序递归重置其被依赖的配置文件，引发重载

gd 仓库支持使用不同的存储底层作为文件来源，并给出了一份本地 `csv` 文件来源的实现，用户可以根据实际情况实现对应的 source，

### 示例：

首先需要定义配置文件的常数，与文件名一一对应：

```go
type ConfigName int

//go:generate stringer -type=ConfigName -trimprefix=Config_
const (
    Config_npc_base = ConfigName(iota)
    Config_npc_appearance
    Config_building
    Max
)

func (c ConfigName) Idx() int {
    return int(c)
}
```

使用 go:generate stringer是为了便于我们从类型定义转换到配置文件的名字，实现Idx函数是为了便于 `gd` 库可以方便地设置入参类型

```go
type Key interface{
    String() string
    Idx() int
}
```

在使用时，需要先初始化 source，然后创建 `gdd` 结构体，注册配置文件的解析函数和依赖并初始化，之后就可以正常地访问配置文件了。

```go
source, _ := gd.NewCsvSource("config")
gdd := gd.NewGdd(Max.Idx(), source)
gdd.Register(Config_npc_appearance, func(s String) interface{}{
    csv := ParseCsv(s)
    doc := make(map[int32]*NpcAppearance)
    unmarshal(doc, csv)
    return doc
}, Config_npc_base)
... // register other config
gdd.Start()

--------

config := gdd.GetDoc(Config_npc_appearance).(map[int32]*NpcAppearance)
use(config)
```

