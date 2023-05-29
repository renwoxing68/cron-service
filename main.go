package main

import (
	"errors"
	"github.com/BurntSushi/toml"
	"github.com/robfig/cron/v3"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type TaskInfoConf struct {
	Name string `toml:"name"`
	Spec string `toml:"spec"`
	Cmd  string `toml:"cmd"`
}

type TaskConf struct {
	Task []TaskInfoConf `toml:"tasks"`
}

var confMap map[string]TaskInfoConf
var EntryMap map[string]cron.EntryID
var c *cron.Cron

func main() {
	taskConf, err := getConf()
	if err != nil {
		log.Fatal("配置读取失败 err:%v", err)
		return
	}
	if len(taskConf.Task) == 0 {
		log.Fatal("无可执行的定时任务")
		return
	}

	for _, conf := range taskConf.Task {
		confMap[conf.Name] = conf
	}
	if len(confMap) != len(taskConf.Task) {
		log.Fatal("任务名称有重复！")
		return
	}

	log.Println("cron Starting...")

	for _, v := range taskConf.Task {
		conf := v
		err = addTask(conf)
		if err != nil {
			log.Printf("任务添加失败：%s,conf:%v,err:%v", conf.Name, conf, err)
			return
		}
	}
	//定时更新配置
	_, err = c.AddFunc("@every 1m", updateConf)
	if err != nil {
		log.Printf("定时更新配置任务失败 err:%v", err)
		return
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

// 增加任务
func addTask(conf TaskInfoConf) error {
	entryID, err := c.AddFunc(conf.Spec, func() {
		logger := initLog()
		//cmd := exec.Command("/bin/bash", "-c", conf.Cmd)
		start := time.Now()
		cmd := exec.Command(conf.Cmd)
		err := cmd.Run()
		cmd.Output()
		if err != nil {
			logger.Printf("cmd run 执行失败：%s,conf:%v,err:%v", conf.Name, conf, err)
			return
		}
		elapsed := time.Since(start)
		logger.Printf("执行success：%s,conf:%v time cost %d", conf.Name, conf, elapsed.Milliseconds())
	})
	if err != nil {
		return err
	}
	EntryMap[conf.Name] = entryID
	return nil
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

	c = cron.New(cron.WithParser(cron.NewParser(
		cron.SecondOptional | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor,
	)))
	//实例化
	confMap = map[string]TaskInfoConf{}
	EntryMap = map[string]cron.EntryID{}
}

// 初始化日志
func initLog() *log.Logger {
	file := "./log/" + time.Now().Format("20060102") + "_cron.log"
	logFile, err := os.OpenFile(file, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0766)
	if err != nil {
		panic(err)
	}
	return log.New(logFile, "[logTool]", log.LstdFlags|log.Lshortfile|log.LUTC)
}

// 读取配置
func getConf() (TaskConf, error) {
	var taskConf TaskConf
	filename := "./conf/conf.toml"
	filePath, err := filepath.Abs(filename)
	if err != nil {
		return taskConf, errors.New("配置有误")
	}
	if _, err = toml.DecodeFile(filePath, &taskConf); err != nil {
		return taskConf, err
	}
	return taskConf, nil
}

// 定时更新conf
func updateConf() {
	taskConf, err := getConf()
	if err != nil {
		log.Fatal("func updateConf 配置读取失败 err:%v", err)
		return
	}
	lastConfMap := map[string]TaskInfoConf{}
	for _, conf := range taskConf.Task {
		lastConfMap[conf.Name] = conf
	}
	if len(lastConfMap) != len(taskConf.Task) {
		log.Fatal("func updateConf 任务名称有重复！")
		return
	}
	var ok bool
	//是否有新增任务
	for _, lastConf := range lastConfMap {
		if _, ok = confMap[lastConf.Name]; !ok { //新增
			err = addTask(lastConf)
			if err != nil {
				log.Printf("fun updateConf 任务添加失败：%s,conf:%v,err:%v", lastConf.Name, lastConf, err)
				return
			}
		} else {
			//配置不变跳过
			if lastConf == confMap[lastConf.Name] {
				continue
			}
			//移除旧任务
			c.Remove(EntryMap[lastConf.Name])
			//添加新任务
			err = addTask(lastConf)
			if err != nil {
				log.Printf("fun updateConf 任务添加失败：%s,conf:%v,err:%v", lastConf.Name, lastConf, err)
				return
			}
		}
	}
	//移除已经删除的任务
	for _, conf := range confMap {
		if _, ok = lastConfMap[conf.Name]; !ok {
			//移除旧任务
			c.Remove(EntryMap[conf.Name])
			// 删除指定的键
			delete(EntryMap, conf.Name)
		}
	}
	confMap = lastConfMap
}
