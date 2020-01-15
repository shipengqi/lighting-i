package main

import "github.com/shipengqi/lighting-i/cmd"

func main()  {
	rootCmd := cmd.NewLightingCommand()
	_ = rootCmd.Execute()
}