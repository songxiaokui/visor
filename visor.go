package main

import (
	"bytes"
	"fmt"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
	"github.com/songxiaokui/visor/config"
	"gopkg.in/gomail.v2"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func visorCpu() {
	for {
		per, _ := cpu.Percent(time.Duration(config.GetConfig().Interval)*time.Second, false)
		if per[0] > config.GetConfig().AlterLimit {
			recordTop10()
		}
	}
}

func visorMem() {
	for {
		m, _ := mem.VirtualMemory()
		if m.UsedPercent > config.GetConfig().AlterLimit {
			recordTop10()
		}
		time.Sleep(time.Duration(config.GetConfig().Interval) * time.Second)
	}
}

func record() {
	now := time.Now().Format("2006-01-02-15-04-05")
	logFile := config.GetConfig().SnapPath + now + ".log"
	f, _ := os.Create(logFile)
	defer f.Close()
	//loadCmd := exec.Command("w")
	//loadOutput, _ := loadCmd.Output()

	f.Write([]byte{'\n'})
	f.Write([]byte{'\n'})
	topCmd := exec.Command("top", "-H", "-w", "512", "-c", "-n", "1", "-b")
	topOutput, _ := topCmd.Output()
	f.Write(topOutput)
	SendMail("你服务器炸了", string(topOutput), config.GetConfig())
}

func stringFormat(psInfo string) string {
	// PID CPU MEM TIME COMMAND
	data := strings.Split(psInfo, "\n")
	outPut := ""
	for i := 0; i < len(data)-1; i++ {
		v := data[i]
		if v == "" {
			continue
		} else {
			for {
				if strings.Contains(v, "  ") {
					v = strings.ReplaceAll(v, "  ", " ")
				} else {
					break
				}
			}
			dataList := strings.Split(v, " ")
			user := dataList[0]
			pid := dataList[1]
			c := dataList[2]
			m := dataList[3]
			command := strings.Join(dataList[10:], " ")
			group := fmt.Sprintf("\t|%s \t|%s \t|%s \t|%s \t|%s \r\n",
				user, pid, c, m, command)
			outPut += group
			outPut += "\r\n"
		}
	}
	return outPut
}

func recordTop10() {
	now := time.Now().Format("2006-01-02-15-04-05")
	logFile := config.GetConfig().SnapPath + now + ".log"
	f, _ := os.Create(logFile)
	defer f.Close()
	// ps aux --sort -rss 输出
	topCmd := exec.Command("ps", "aux", "--sort=-rss")
	headCmd := exec.Command("head", "-10")
	var outPutTop bytes.Buffer
	topCmd.Stdout = &outPutTop
	if err := topCmd.Start(); err != nil {
		return
	}
	if err := topCmd.Wait(); err != nil {
		return
	}
	// 输入
	headCmd.Stdin = &outPutTop
	var headCmdOut bytes.Buffer
	headCmd.Stdout = &headCmdOut

	if err := headCmd.Start(); err != nil {
		return
	}
	if err := headCmd.Wait(); err != nil {
		return
	}
	// 写入邮件中排名前10的进程
	out := stringFormat(string(headCmdOut.Bytes()))
	m, _ := mem.VirtualMemory()
	subject := fmt.Sprintf("农林大学-生产环境: 爆炸了！内存: %.2f", m.UsedPercent)
	SendMail(subject, out, config.GetConfig())
	f.Write([]byte(out))
}

func SendMail(subject string, body string, conf config.Config) error {
	mailConn := map[string]string{
		"user": conf.FromMail,
		"pass": conf.FromMailPass,
		"host": conf.FromMailHost,
		"port": conf.FromMailPort,
	}

	port, _ := strconv.Atoi(mailConn["port"])

	m := gomail.NewMessage()
	m.SetHeader("From", "<"+mailConn["user"]+">") //这种方式可以添加别名，即“XD Game”， 也可以直接用<code>m.SetHeader("From",mailConn["user"])</code> 读者可以自行实验下效果
	m.SetHeader("To", conf.ToMail...)             //发送给多个用户
	m.SetHeader("Subject", subject)               //设置邮件主题
	m.SetBody("text/html", body)                  //设置邮件正文

	d := gomail.NewDialer(mailConn["host"], port, mailConn["user"], mailConn["pass"])

	err := d.DialAndSend(m)
	return err
}
