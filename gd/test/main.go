package main

import (
	"fmt"
	"time"

	"github.com/legamerdc/game_tools/gd"
)

func main() {
	source, e := gd.NewCsvSource("./")
	if e != nil {
		fmt.Println(e)
		return
	}
	gdd := gd.NewGdd(Max.Idx(), source)
	gdd.Register(CfgTest, func(s string) interface{} {
		fmt.Println(s)
		return nil
	})
	gdd.Start()
	for {
		_ = gdd.GetDoc(CfgTest)
		time.Sleep(time.Second * 10)
	}
}
