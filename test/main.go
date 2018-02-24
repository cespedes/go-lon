package main

import (
	"fmt"
	lon "../"
)

func main() {
	fmt.Println("Hello LON")
	c, err := lon.Dial("192.168.1.133")
	if err != nil {
		fmt.Println(err)
	}
	for n:=0; n<10; n++ {
		p, e := c.Read()
		if e != nil {
			fmt.Println(e)
		} else {
			fmt.Println(p)
		}
	}
	c.Close()
}
