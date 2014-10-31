package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
)

type file struct {
	FileName string
	isIni    bool
	File     *[][]byte
}

func main() {
	if len(os.Args) == 1 {
		fmt.Println(`这是一个可以处理从VCR里导出的txt文档的工具，同时可以对ini文档作一些处理。
具体功能和用法如下：

1、处理从VCR里导出的txt文档：

	请将转出的txt文件拖到本程序上，本程序将自动根据文本判断出是CAN还是非CAN的文件（默认非CAN），将生成名为 原文件名_M.ini （Modifiled）的文件在同一目录下。

2、处理ini（BMW CAN、BMW UDS）文件：
	可处理如下情况：

		末尾多余00
		RX,6F1,8,40 02 1A 80 00 00 00 00,

		TX,6F1情况
		TX,6F1,4,40 30 00 00,

3、本工具支持多文件处理。


有任何问题，联系 Wei.Wei@snapon.com
按<Enter>退出`)
		fmt.Scanln()
		return
	}
	filelist := GetFiles()
	for i, _ := range *filelist {
		if (*filelist)[i].isIni {
			(*filelist)[i].File = trimTX6F1((*filelist)[i].File)
			iniTrimBellyfat((*filelist)[i].File)
			ioutil.WriteFile((*filelist)[i].FileName+"_M.ini", bytes.Join(*((*filelist)[i].File), []byte{13, 10}), 0666)
		} else {
			if vcrIsCAN((*filelist)[i].File) {
				(*filelist)[i].File = vcrCAN((*filelist)[i].File)
				(*filelist)[i].File = trimTX6F1((*filelist)[i].File)
				iniTrimBellyfat((*filelist)[i].File)
				ioutil.WriteFile((*filelist)[i].FileName+"_M.ini", bytes.Join(*((*filelist)[i].File), []byte{13, 10}), 0666)
			} else {
				(*filelist)[i].File = vcrNoCAN((*filelist)[i].File)
				ioutil.WriteFile((*filelist)[i].FileName+"_M.rns", bytes.Join(*((*filelist)[i].File), []byte{13, 10}), 0666)
			}
		}
	}
}

//获取将要处理的文件列表，文件为拖到程序上的所有ini和txt文件
func GetFiles() *[]file {
	fileNames := os.Args[1:]
	files := []file{}
	for _, fileName := range fileNames {
		if strings.HasSuffix(fileName, ".ini") {
			var f file
			f.FileName = fileName[:len(fileName)-4]
			f.isIni = true
			file, err := ioutil.ReadFile(fileName)
			if err != nil {
				log.Println("文件", fileName, "读取出错:\n")
				log.Println(err)
			}
			File := (bytes.Split(file, []byte("\r\n")))
			for i, _ := range File {
				File[i] = bytes.TrimSuffix(File[i], []byte(" "))
			}
			f.File = &File
			files = append(files, f)
		} else if strings.HasSuffix(fileName, ".txt") {
			var f file
			f.FileName = fileName[:len(fileName)-4]
			f.isIni = false
			file, err := ioutil.ReadFile(fileName)
			if err != nil {
				log.Println("文件", fileName, "读取出错:\n")
				log.Println(err)
			}
			File := (bytes.Split(file, []byte("\r\n")))
			for i, _ := range File {
				File[i] = bytes.TrimSuffix(File[i], []byte(" "))
			}
			f.File = &File
			files = append(files, f)
		}
	}
	return &files
}

func vcrIsCAN(lines *[][]byte) bool {
	for _, line := range *lines {
		if string(line[30:33]) == "CAN" && string(line[27:29]) == "RX" {
			return true
		}
	}
	return false
}

func vcrCAN(lines *[][]byte) *[][]byte {
	var (
		lastTX bool
		out    [][]byte
	)
	for _, line := range *lines {
		if string(line[30:33]) == "CAN" && string(line[27:29]) == "TX" {
			if lastTX {
				out = append(out, []byte{})
			}
			tmp := append([]byte{82, 88, 44, line[36], line[37], line[38], 44, line[42], 44}, line[45:]...)
			out = append(out, append(tmp, 44))

			lastTX = false
		} else if string(line[30:33]) == "CAN" && string(line[27:29]) == "RX" {
			tmp := append([]byte{84, 88, 44, line[36], line[37], line[38], 44, line[42], 44}, line[45:]...)
			out = append(out, append(tmp, 44))
			lastTX = true
		} else {
			if len(line) > 22 {
				lastTX = false
				out = append(out, append([]byte(";"), line[21:]...))
			}
			lastTX = false
		}
	}
	return &out
}
func vcrNoCAN(lines *[][]byte) *[][]byte {
	var (
		lastline int
		out      [][]byte
	)
	out = append(out, []byte{80, 114, 111, 116, 111, 99, 111, 108, 58, 48, 59, 9, 9, 9, 9, 9, 9, 47, 47, 48, 58, 75, 87, 50, 48, 48, 48, 44, 32, 49, 58, 68, 83, 50, 44, 32, 50, 58, 83, 73, 78, 71, 76, 69, 44, 32, 51, 58, 73, 83, 79, 44, 32, 56, 58, 79, 84, 72, 69, 82})
	out = append(out, []byte{66, 121, 116, 101, 70, 111, 114, 109, 97, 116, 58, 78, 95, 56, 95, 48, 95, 65, 59, 9, 9, 9, 9, 47, 47, 79, 124, 69, 124, 78, 95, 55, 124, 56, 95, 48, 124, 49, 124, 50, 95, 65, 124, 88, 124, 78, 124, 84, 44, 183, 214, 177, 240, 177, 237, 202, 190, 163, 186, 198, 230, 197, 188, 206, 222, 208, 163, 209, 233, 161, 162, 202, 253, 190, 221, 206, 187, 179, 164, 182, 200, 161, 162, 205, 163, 214, 185, 206, 187, 179, 164, 182, 200, 161, 162, 65, 68, 68, 186, 205, 88, 79, 82, 186, 205, 195, 187, 211, 208, 202, 253, 190, 221, 176, 252, 208, 163, 209, 233})
	out = append(out, []byte{77, 115, 76, 101, 110, 58, 48, 48, 59, 9, 9, 9, 9, 9, 9, 47, 47, 181, 218, 210, 187, 184, 246, 215, 214, 183, 251, 206, 170, 177, 237, 202, 190, 202, 253, 190, 221, 176, 252, 181, 196, 179, 164, 182, 200, 181, 196, 206, 187, 214, 195, 163, 172, 180, 211, 49, 191, 170, 202, 188, 163, 187, 181, 218, 182, 254, 184, 246, 215, 214, 183, 251, 206, 170, 48, 177, 237, 202, 190, 184, 249, 190, 221, 80, 114, 111, 116, 111, 99, 111, 108, 197, 208, 182, 207, 163, 172, 206, 170, 49, 177, 237, 202, 190, 200, 161, 181, 205, 203, 196, 206, 187, 163, 172, 206, 170, 50, 177, 237, 202, 190, 200, 161, 184, 223, 203, 196, 206, 187, 163, 172, 206, 170, 51, 177, 237, 202, 190, 200, 161, 56, 206, 187, 163, 187, 200, 231, 163, 186, 50, 51, 177, 237, 202, 190, 213, 251, 184, 246, 181, 218, 182, 254, 206, 187, 206, 170, 179, 164, 182, 200, 206, 187})
	out = append(out, []byte{73, 115, 65, 117, 116, 111, 82, 101, 115, 112, 111, 110, 115, 101, 58, 48, 59, 9, 9, 9, 9, 47, 47, 206, 170, 49, 177, 237, 202, 190, 200, 231, 185, 251, 195, 187, 211, 208, 178, 233, 213, 210, 181, 189, 202, 253, 190, 221, 176, 252, 215, 212, 182, 175, 187, 216, 184, 180})
	out = append(out, []byte{65, 100, 100, 114, 101, 115, 115, 119, 111, 114, 100, 58, 48, 59, 9, 9, 9, 9, 9, 47, 47, 200, 231, 185, 251, 206, 170, 48, 177, 237, 202, 190, 191, 236, 203, 217, 179, 245, 202, 188, 187, 175})
	out = append(out, []byte{73, 110, 105, 116, 105, 97, 108, 105, 122, 101, 58, 68, 79, 56, 70, 55, 48, 70, 57, 59, 9, 9, 9, 47, 47, 200, 231, 185, 251, 206, 170, 194, 253, 203, 217, 179, 245, 202, 188, 187, 175, 163, 172, 211, 201, 215, 243, 214, 193, 211, 210, 183, 214, 177, 240, 206, 170, 163, 186, 75, 69, 89, 66, 89, 84, 69, 49, 44, 75, 69, 89, 66, 89, 84, 69, 50, 44, 126, 75, 69, 89, 66, 89, 84, 69, 50, 44, 126, 65, 100, 100, 114, 101, 115, 115, 119, 111, 114, 100})
	out = append(out, []byte{66, 97, 117, 100, 114, 97, 116, 101, 58, 49, 49, 53, 50, 48, 48, 59, 9, 9, 9, 9, 47, 47, 178, 168, 204, 216, 194, 202})
	out = append(out, []byte{70, 105, 114, 115, 116, 66, 121, 116, 101, 58, 56, 50, 44, 56, 51, 44, 67, 50, 44, 56, 53, 44, 67, 52, 44, 56, 54, 59, 9, 47, 47, 191, 236, 203, 217, 179, 245, 202, 188, 187, 175, 191, 201, 196, 220, 181, 196, 181, 218, 210, 187, 184, 246, 66, 89, 84, 69, 44, 215, 238, 182, 224, 178, 187, 179, 172, 185, 253, 49, 48, 184, 246})
	out = append(out, []byte{})
	out = append(out, []byte{})
	for _, line := range *lines {
		if string(line[30:34]) == "UART" && string(line[27:29]) == "TX" {
			if lastline == 0 {
				lastline = 1
				out = append(out, append([]byte(">,"), line[36], line[37]))
			} else if lastline == 2 {
				lastline = 1
				out = append(out, []byte{})
				out = append(out, append([]byte(">,"), line[36], line[37]))
			} else if lastline == 1 {
				out[len(out)-1] = append(out[len(out)-1], byte(','), line[36], line[37])
			} else {
				out = append(out, append([]byte("//"), line...))
			}
		} else if string(line[30:34]) == "UART" && string(line[27:29]) == "RX" {
			if lastline == 1 {
				lastline = 2
				out = append(out, append([]byte("<,"), line[36], line[37]))
			} else if lastline == 2 {
				out[len(out)-1] = append(out[len(out)-1], byte(','), line[36], line[37])
			} else {
				out = append(out, append([]byte("//"), line...))
			}
		} else if len(line) > 22 {
			lastline = 0
			out = append(out, append([]byte("//"), line[21:]...))
		}
	}
	return &out
}

func trimTX6F1(lines *[][]byte) *[][]byte {
	var out [][]byte
	for _, line := range *lines {
		if !bytes.HasPrefix(line, []byte("TX,6F1,")) {
			out = append(out, line)
		}
	}
	return &out
}

//对ini文件末尾多余的00之类的东西进行去除
//RX,6F1,8,40 02 1A 80 00 00 00 00,
func iniTrimBellyfat(lines *[][]byte) {
	//对读取的文件分行
	for i, line := range *lines {
		//当前行是否是发包行
		if bytes.HasPrefix(line, []byte("RX")) {
			//获取将当前行实际长度"02"
			t, err := strconv.ParseInt(string(line[12:14]), 16, 32)
			if err != nil {
				continue
			}
			//检查是否是短包
			if 0 < t && t < 6 {
				//对短包截取前半截“RX,6F1,8,40 02 1A 80”+“,”
				line = append(line[:14+3*t], byte(','))
				//修改包长"8"->"02"+2
				line[7] = byte(t + 0x32)
				//对“30 00 00”的处理
			} else if t == 0x30 {
				line = append(line[:20], byte(','))
				line[7] = byte('4')
			}
			(*lines)[i] = line //修改的行写入原始数据
		}
	}
}