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
	b := make([]byte, 4096)
	for n:=0; n<10; n++ {
		i, e, cnip := c.Read(b)
		if e != nil {
			fmt.Println(e)
		} else {
			fmt.Println(b[0:i])
			fmt.Println(cnip)
		}
	}
	c.Close()
}
