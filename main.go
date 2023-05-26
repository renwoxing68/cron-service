package main

import (
	"github.com/BurntSushi/toml"
	"github.com/robfig/cron/v3"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

func main() {
	type TaskInfoConf struct {
		Name string `toml:"name"`
		Spec string `toml:"spec"`
		Cmd  string `toml:"cmd"`
	}
	type TaskConf struct {
		Task []TaskInfoConf `toml:"tasks"`
	}

	filename := "./conf/conf.toml"
	filePath, err := filepath.Abs(filename)
	if err != nil {
		log.Fatal("配置有误")
		return
	}

	var taskConf TaskConf
	if _, err = toml.DecodeFile(filePath, &taskConf); err != nil {
		log.Fatal(err)
	}

	if len(taskConf.Task) == 0 {
		log.Fatal("无可执行的定时任务")
		return
	}

	log.Println("cron Starting...")
	c := cron.New(cron.WithParser(cron.NewParser(
		cron.SecondOptional | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor,
	)))

	for _, v := range taskConf.Task {
		conf := v
		_, _ = c.AddFunc(conf.Spec, func() {
			logger := initLog()
			cmd := exec.Command("/bin/bash", "-c", conf.Cmd)
			e := cmd.Run()
			logger.Printf("任务：%s,conf:%v,err:%v", conf.Name, conf, e)
		})
	}

	c.Start()
	t1 := time.NewTimer(time.Second * 10)
	for {
		select {
		case <-t1.C:
			t1.Reset(time.Second * 10)
		}
	}
}

func init() {
	directory := "./log"
	// 检查目录是否存在
	_, err := os.Stat(directory)
	if os.IsNotExist(err) {
		// 目录不存在，创建它
		err = os.Mkdir(directory, 0755)
		if err != nil {
			panic("无法创建log目录:")
		}
	}

}

func initLog() *log.Logger {
	file := "./log/" + time.Now().Format("20060102") + "_cron.log"
	logFile, err := os.OpenFile(file, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0766)
	if err != nil {
		panic(err)
	}
	return log.New(logFile, "[logTool]", log.LstdFlags|log.Lshortfile|log.LUTC)
}
