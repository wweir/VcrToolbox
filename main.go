package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
)

type file struct {
	FileName string
	fileType int // 1.txt; 2.ini; 3.rns; 4._M.ini; 5._M.rns; 6.(bmw)stxt(txt); 7.CANSIMLog file; 8.Libra(REC) KW2000
	File     *[][]byte
}

const (
	txt     = 1
	ini     = 2
	rns     = 3
	Mini    = 4
	Mrns    = 5
	stxt    = 6
	CanSim  = 7
	LibraKW = 8
)

func main() {
	if len(os.Args) == 1 {
		fmt.Println(`这是一个可以处理从VCR里导出的txt文档的工具，同时可以对ini文档作一些处理。
具体功能和用法如下：

1、处理从VCR里导出的txt文档：

	请将转出的txt文件拖到本工具上，本工具将自动根据文本内容生成对应 Simulator 可用的 ini 或 rns 文件，将生成名为 原文件名_M.ini (rns, M->Modified) 的文件在同一目录下。此次输出的文件包含VCR中所有包(未去重)，并且带有 VCR 中的文本注释。

	再次将 原文件名_M.ini(rns) 文件拖到本工具上，将删除里面所有重复包和注释文本，生成 原文件名_MM.ini (rns) 文件。如想删除其它原有的 ini/rns 文件中的重复包，请自行为文件名加上 _M 后缀。

2、处理BMW CAN/UDS/ComWatch/CANSIMLog 文件：
	可处理如下情况：

		末尾多余00
		RX,6F1,8,40 02 1A 80 00 00 00 00,

		长度位多0
		TX,663,06,F1 04 71 01 0F 0C,

3、本工具支持多文件处理。

如有问题，请尝试最新版本  https://github.com/wweir/BMW_Toolbox
仍有问题或有其它需求，联系  Wei.Wei@snapon.com
按<Enter>退出`)
		fmt.Println(os.Args[0] + "/Header.rns")
		fmt.Scanln()
		return
	}
	filelist := GetFiles()
	for i, _ := range *filelist {
		switch (*filelist)[i].fileType {
		case txt:
			out, isOrNot := vcrCAN((*filelist)[i].File)
			if isOrNot {
				out = trimTX6F1(out)
				ioutil.WriteFile((*filelist)[i].FileName+"_M.ini", bytes.Join(*out, []byte{13, 10}), 0666)
			}

			out, isOrNot = vcrNoCAN((*filelist)[i].File)
			if isOrNot {
				ioutil.WriteFile((*filelist)[i].FileName+"_M.rns", bytes.Join(*out, []byte{13, 10}), 0666)
			}
		case ini:
			out := trimTX6F1((*filelist)[i].File)

			lengthError(out)
			iniTrimBellyfat(out)

			ioutil.WriteFile((*filelist)[i].FileName+"_M.ini", bytes.Join(*out, []byte{13, 10}), 0666)
		case Mini:
			out := getPackages((*filelist)[i].File)
			out = DeleteRepeat(out)

			ioutil.WriteFile((*filelist)[i].FileName+"_MM.ini", bytes.Join(*out, []byte{13, 10}), 0666)
		case Mrns:
			out := getPackages((*filelist)[i].File)
			out = DeleteRepeat(out)

			outFile := append((*((*filelist)[i].File))[:9], *out...)
			ioutil.WriteFile((*filelist)[i].FileName+"_MM.rns", bytes.Join(outFile, []byte{13, 10}), 0666)
		case stxt:
			out := ComWatch((*filelist)[i].File)
			ioutil.WriteFile((*filelist)[i].FileName+"_CM.rns", bytes.Join(*out, []byte{13, 10}), 0666)
		case CanSim:
			out := CanSimLog((*filelist)[i].File)
			ioutil.WriteFile((*filelist)[i].FileName+".ini", bytes.Join(*out, []byte{13, 10}), 0666)
		case LibraKW:
			out := recKWHS((*filelist)[i].File)
			ioutil.WriteFile((*filelist)[i].FileName+"_REC.rns", bytes.Join(*out, []byte{13, 10}), 0666)
		}
	}
}

//获取将要处理的文件列表，文件为拖到程序上的所有ini和txt文件
func GetFiles() *[]file {
	fileNames := os.Args[1:]
	files := []file{}
	for _, fileName := range fileNames {
		if bytes.HasSuffix([]byte(fileName), []byte(".txt")) || bytes.HasSuffix([]byte(fileName), []byte(".TXT")) {
			var f file

			f.FileName = fileName[:len(fileName)-4]

			file, err := ioutil.ReadFile(fileName)
			if err != nil {
				log.Println("文件", fileName, "读取出错:\n")
				log.Println(err)
			}
			File := (bytes.Split(file, []byte("\r\n")))
			out := [][]byte{}

			//For .stxt in .txt suffix
			if bytes.HasPrefix(File[1], []byte(";ComWatch ")) {
				f.fileType = stxt
				out = File

			} else if bytes.HasSuffix([]byte(fileName), []byte("CANSIMLog.txt")) {
				f.fileType = CanSim
				out = File

			} else {
				f.fileType = txt

				for i, _ := range File {
					File[i] = bytes.TrimSuffix(File[i], []byte(" "))
					if len(File[i]) > 22 {
						out = append(out, File[i][21:])
					}
				}
			}
			f.File = &out
			files = append(files, f)

		} else if bytes.HasSuffix([]byte(fileName), []byte(".stxt")) {
			var f file

			f.FileName = fileName[:len(fileName)-6]
			f.fileType = stxt

			file, err := ioutil.ReadFile(fileName)
			if err != nil {
				log.Println("文件", fileName, "读取出错:\n")
				log.Println(err)
			}

			out := (bytes.Split(file, []byte("\r\n")))
			f.File = &out

			files = append(files, f)

		} else if bytes.HasSuffix([]byte(fileName), []byte(".REC")) {
			var f file
			f.FileName = fileName[:len(fileName)-4]
			f.fileType = LibraKW

			out, err := ioutil.ReadFile(fileName)
			if err != nil {
				log.Println("文件", fileName, "读取出错:\n")
				log.Println(err)
			}

			f.File = &([][]byte{out})

			files = append(files, f)

		} else {
			var (
				f        file
				iniOrRns bool //判断是否是ini或者rns文件
			)
			if bytes.HasSuffix([]byte(fileName), []byte("_M.ini")) {
				f.FileName = fileName[:len(fileName)-6]
				f.fileType = Mini
				iniOrRns = true

			} else if bytes.HasSuffix([]byte(fileName), []byte("_M.rns")) {
				f.FileName = fileName[:len(fileName)-6]
				f.fileType = Mrns
				iniOrRns = true

			} else if bytes.HasSuffix([]byte(fileName), []byte(".ini")) || bytes.HasSuffix([]byte(fileName), []byte(".INI")) {
				f.FileName = fileName[:len(fileName)-4]
				f.fileType = ini
				iniOrRns = true
			}

			if iniOrRns {
				file, err := ioutil.ReadFile(fileName)
				if err != nil {
					log.Println("文件", fileName, "读取出错:\n")
					log.Println(err)
					continue
				}

				File := (bytes.Split(file, []byte("\r\n")))
				f.File = &File

				files = append(files, f)
			}
		}
	}
	return &files
}

//BMW KWHS Header
func BMW_KWHS_Header() [][]byte {
	return [][]byte{
		{80, 114, 111, 116, 111, 99, 111, 108, 58, 48, 59, 9, 9, 9, 47, 47, 48, 58, 75, 87, 50, 48, 48, 48, 44, 32, 49, 58, 68, 83, 50, 44, 32, 50, 58, 83, 73, 78, 71, 76, 69, 44, 32, 51, 58, 73, 83, 79, 44, 32, 56, 58, 79, 84, 72, 69, 82},
		{66, 121, 116, 101, 70, 111, 114, 109, 97, 116, 58, 78, 95, 56, 95, 48, 95, 65, 59, 9, 9, 47, 47, 79, 124, 69, 124, 78, 95, 55, 124, 56, 95, 48, 124, 49, 124, 50, 95, 65, 124, 88, 124, 78, 124, 84, 44, 183, 214, 177, 240, 177, 237, 202, 190, 163, 186, 198, 230, 197, 188, 206, 222, 208, 163, 209, 233, 161, 162, 202, 253, 190, 221, 206, 187, 179, 164, 182, 200, 161, 162, 205, 163, 214, 185, 206, 187, 179, 164, 182, 200, 161, 162, 65, 68, 68, 186, 205, 88, 79, 82, 186, 205, 195, 187, 211, 208, 202, 253, 190, 221, 176, 252, 208, 163, 209, 233},
		{77, 115, 76, 101, 110, 58, 48, 48, 59, 9, 9, 9, 47, 47, 181, 218, 210, 187, 184, 246, 215, 214, 183, 251, 206, 170, 177, 237, 202, 190, 202, 253, 190, 221, 176, 252, 181, 196, 179, 164, 182, 200, 181, 196, 206, 187, 214, 195, 163, 172, 180, 211, 49, 191, 170, 202, 188, 163, 187, 181, 218, 182, 254, 184, 246, 215, 214, 183, 251, 206, 170, 48, 177, 237, 202, 190, 184, 249, 190, 221, 80, 114, 111, 116, 111, 99, 111, 108, 197, 208, 182, 207, 163, 172, 206, 170, 49, 177, 237, 202, 190, 200, 161, 181, 205, 203, 196, 206, 187, 163, 172, 206, 170, 50, 177, 237, 202, 190, 200, 161, 184, 223, 203, 196, 206, 187, 163, 172, 206, 170, 51, 177, 237, 202, 190, 200, 161, 56, 206, 187, 163, 187, 200, 231, 163, 186, 50, 51, 177, 237, 202, 190, 213, 251, 184, 246, 181, 218, 182, 254, 206, 187, 206, 170, 179, 164, 182, 200, 206, 187},
		{73, 115, 65, 117, 116, 111, 82, 101, 115, 112, 111, 110, 115, 101, 58, 48, 59, 9, 9, 47, 47, 206, 170, 49, 177, 237, 202, 190, 200, 231, 185, 251, 195, 187, 211, 208, 178, 233, 213, 210, 181, 189, 202, 253, 190, 221, 176, 252, 215, 212, 182, 175, 187, 216, 184, 180},
		{65, 100, 100, 114, 101, 115, 115, 119, 111, 114, 100, 58, 48, 59, 9, 9, 9, 47, 47, 200, 231, 185, 251, 206, 170, 48, 177, 237, 202, 190, 191, 236, 203, 217, 179, 245, 202, 188, 187, 175},
		{73, 110, 105, 116, 105, 97, 108, 105, 122, 101, 58, 68, 79, 56, 70, 55, 48, 70, 57, 59, 9, 9, 47, 47, 200, 231, 185, 251, 206, 170, 194, 253, 203, 217, 179, 245, 202, 188, 187, 175, 163, 172, 211, 201, 215, 243, 214, 193, 211, 210, 183, 214, 177, 240, 206, 170, 163, 186, 75, 69, 89, 66, 89, 84, 69, 49, 44, 75, 69, 89, 66, 89, 84, 69, 50, 44, 126, 75, 69, 89, 66, 89, 84, 69, 50, 44, 126, 65, 100, 100, 114, 101, 115, 115, 119, 111, 114, 100}, {66, 97, 117, 100, 114, 97, 116, 101, 58, 49, 49, 53, 50, 48, 48, 59, 9, 9, 47, 47, 178, 168, 204, 216, 194, 202}, {70, 105, 114, 115, 116, 66, 121, 116, 101, 58, 56, 50, 44, 56, 51, 44, 67, 50, 44, 56, 53, 44, 67, 52, 44, 56, 54, 59, 9, 47, 47, 191, 236, 203, 217, 179, 245, 202, 188, 187, 175, 191, 201, 196, 220, 181, 196, 181, 218, 210, 187, 184, 246, 66, 89, 84, 69, 44, 215, 238, 182, 224, 178, 187, 179, 172, 185, 253, 49, 48, 184, 246},
		{},
		{}}
}

//取出独立完整的包(发包+回包)
func getPackages(lines *[][]byte) *[][]byte {
	var (
		out           [][]byte
		lastIsPackage bool
	)
	for _, line := range *lines {
		if bytes.HasPrefix(line, []byte("RX,")) || bytes.HasPrefix(line, []byte(">,")) {
			//中继包 30
			if bytes.Contains(line, []byte(",30 ")) || (bytes.HasPrefix(line, []byte("TX,6F1,")) && (line[12] == 33 && line[13] == 30)) {
				out[len(out)-1] = append(out[len(out)-1], 13, 10, 13, 10) //加入换行
				out[len(out)-1] = append(out[len(out)-1], line...)
			} else {
				out = append(out, append([]byte{13, 10}, line...))
			}
			lastIsPackage = true

		} else if bytes.HasPrefix(line, []byte("TX,")) || bytes.HasPrefix(line, []byte("<,")) {
			if lastIsPackage {
				out[len(out)-1] = append(out[len(out)-1], 13, 10) //加入换行
				out[len(out)-1] = append(out[len(out)-1], line...)
				//中继包 21
			} else if bytes.Contains(line, []byte(",21 ")) || bytes.Contains(line, []byte(",F1 21 ")) {
				out[len(out)-1] = append(out[len(out)-1], 13, 10, 13, 10) //加入换行
				out[len(out)-1] = append(out[len(out)-1], line...)
			} else {
				out[len(out)-1] = append(out[len(out)-1], 13, 10, 13, 10) //加入换行
				out[len(out)-1] = append(out[len(out)-1], line...)
			}
			lastIsPackage = true

		} else {
			lastIsPackage = false
		}
	}
	return &out
}

//去除重复包
func DeleteRepeat(lines *[][]byte) *[][]byte {
	var (
		out      [][]byte
		isRepeat bool
	)
	for _, line := range *lines {
		isRepeat = false
		for i := range out {
			if bytes.Equal(line, out[i]) {
				isRepeat = true
			}
		}
		if !isRepeat {
			out = append(out, line)
		}
	}
	return &out
}

//For Libra KWHS BMW
func recKWHS(all *[][]byte) *[][]byte {
	var (
		bin      []byte
		rns      [][]byte
		TS       int //TS(ms)
		TSL      int //TSLast
		line     []byte
		thisByte [2]byte
		err00    bool
	)
	devide := bytes.Index((*all)[0], []byte{0x2a, 0x0d, 0x0a, 0x3a, 0x3a, 0x3a, 0x3a, 0x0d, 0x0a})

	rns = BMW_KWHS_Header()
	header := bytes.Split((*all)[0][:devide], []byte{13, 10})
	for i := 17; i < 29; i++ {
		rns = append(rns, append([]byte("//"), header[i]...))
	}

	bin = (*all)[0][devide+10:]
	lenBin := len(bin) - 4
	for i := 0; i < lenBin; i += 9 {
		TS = int(bin[i])<<16 + int(bin[i+1])<<8 + int(bin[i+2])

		if bin[i+4]&0xF < 10 {
			thisByte[1] = bin[i+4]&0xF + 0x30
		} else {
			thisByte[1] = bin[i+4]&0xF + 0x37
		}
		if (bin[i+4]&0xF0)>>4 < 10 {
			thisByte[0] = (bin[i+4]&0xF0)>>4 + 0x30
		} else {
			thisByte[0] = (bin[i+4]&0xF0)>>4 + 0x37
		}

		if TS-TSL > 25 && lenBin > i+20 {
			err00 = false
			rns = append(rns, line)
			if bin[i+22] == 0xf1 {
				line = append([]byte("\r\n>,"), thisByte[0], thisByte[1])
			} else if bin[i+13] == 0xf1 {
				line = append([]byte("<,"), thisByte[0], thisByte[1])
			} else if bin[i+4] == 0 {
				err00 = true
			} else {
				line = append([]byte("//ERROR LINE\r\n//,"), thisByte[0], thisByte[1])
			}
		} else if err00 {
		} else {
			line = append(line, byte(','), thisByte[0], thisByte[1])
		}
		TSL = TS
	}
	rns = append(rns, line)
	return &rns
}

//ComWatch log file
func ComWatch(lines *[][]byte) *[][]byte {
	//0 发包；1 回包；2 其余
	var lastStatus int
	out := BMW_KWHS_Header()
	for i := range (*lines)[6:] {
		i += 6
		//发包 >,
		if len((*lines)[i]) > 20 && string((*lines)[i][16:18]) == "F1" {
			tmp := bytes.Replace((*lines)[i][10:], []byte{0x20}, []byte{0x2C}, -1)
			tmp = tmp[:len(tmp)-1]
			if lastStatus == 1 || (lastStatus == 0 && (string(out[len(out)-1]) != ">,"+string(tmp))) {
				out = append(out, []byte{})
			}
			out = append(out, append([]byte(">,"), tmp...))
			lastStatus = 0

			//回包 <,
		} else if len((*lines)[i]) > 20 && string((*lines)[i][13:15]) == "F1" {
			tmp := bytes.Replace((*lines)[i][10:], []byte{0x20}, []byte{0x2C}, -1)
			tmp = tmp[:len(tmp)-1]
			out = append(out, append([]byte("<,"), tmp...))
			lastStatus = 1

		} else {
			out = append(out, append([]byte("//"), ((*lines)[i])...))
			lastStatus = 2
		}
	}
	return &out
}

//CAN VCR处理
func vcrCAN(lines *[][]byte) (*[][]byte, bool) {
	var (
		exist  bool
		lastTX bool
		out    [][]byte
	)
	for _, line := range *lines {
		if string(line[9:12]) == "CAN" && string(line[6:8]) == "TX" {
			if lastTX {
				out = append(out, []byte{})
			}
			end := bytes.IndexByte(line, byte(']'))
			//Add RX,
			tmp := append([]byte{82, 88, 44}, line[15:end]...)
			tmp = append(tmp, 44, line[end+3], 44)
			tmp = append(tmp, line[end+6:]...)
			out = append(out, append(tmp, 44))

			lastTX = false
		} else if string(line[9:12]) == "CAN" && string(line[6:8]) == "RX" {

			exist = true
			end := bytes.IndexByte(line, byte(']'))
			//Add TX,
			tmp := append([]byte{84, 88, 44}, line[15:end]...)
			tmp = append(tmp, 44, line[end+3], 44)
			tmp = append(tmp, line[end+6:]...)
			out = append(out, append(tmp, 44))

			lastTX = true
		} else {
			lastTX = false
			out = append(out, append([]byte(";"), line...))
		}
	}
	return &out, exist
}

//非CAN VCR处理
func vcrNoCAN(lines *[][]byte) (*[][]byte, bool) {
	var (
		exsit      bool
		lastLine   []byte
		lineStatus int
		out        [][]byte
	)
	out = BMW_KWHS_Header()
	for _, line := range *lines {
		if string(line[9:13]) == "UART" && string(line[6:8]) == "TX" {
			if lineStatus == 2 {
				out = append(out, lastLine)
				out = append(out, []byte{})
			}
			if len(line) > 17 {
				t := bytes.Split(line, []byte(" "))

				tmp := bytes.Join(t[8:], []byte(","))
				lastLine = append([]byte(">,"), tmp...)

			} else if lineStatus != 1 {
				lastLine = append([]byte(">,"), line[15], line[16])

			} else {
				lastLine = append(lastLine, byte(0x2C), line[15], line[16])
			}
			lineStatus = 1

		} else if string(line[9:13]) == "UART" && string(line[6:8]) == "RX" {
			if lineStatus == 1 {
				//针对夹花重复的情况
				//      TX UART: 81
				//      TX UART: 81
				if lastLine[len(lastLine)-2] != line[15] || lastLine[len(lastLine)-1] != line[16] {
					out = append(out, lastLine)

					lastLine = append([]byte("<,"), line[15], line[16])
					lineStatus = 2
				}
				//处理单独的00回包的情况
			} else if lineStatus == 0 {
				if !bytes.HasSuffix(line, []byte("RX UART: 00")) {
					lastLine = append([]byte("<,"), line[15], line[16])
					lineStatus = 2
				}
			} else {
				lastLine = append(lastLine, byte(0x2C), line[15], line[16])
				lineStatus = 2
				exsit = true
			}
		} else {
			if lineStatus != 0 {
				out = append(out, lastLine)
			}
			lineStatus = 0
			out = append(out, append([]byte("//"), line...))
		}
	}
	return &out, exsit
}

//去除TX,6F1,
func trimTX6F1(lines *[][]byte) *[][]byte {
	var (
		out      [][]byte
		lastline string
	)
	for i := range *lines {
		if bytes.HasPrefix((*lines)[i], []byte("TX,")) {
			//解决如下情形：
			//RX,6F1,4,40 30 00 00,
			//TX,6F1,4,40 30 00 00,
			if string((*lines)[i][1:]) != lastline {
				out = append(out, (*lines)[i])
			}

		} else {
			out = append(out, (*lines)[i])
		}

		if len((*lines)[i]) > 2 {
			lastline = string((*lines)[i][1:])

		} else {
			lastline = ""
		}
	}
	return &out
}

//CAM SIM log file
func CanSimLog(lines *[][]byte) *[][]byte {
	var (
		out      [][]byte
		lastIsTX bool
	)
	for i := range *lines {
		//去除多余空格  RX,6F1,4,72 30 00 00    ,
		tmp := bytes.Replace((*lines)[i], []byte{0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x2C}, []byte{0x2C}, -1)
		tmp = bytes.Replace(tmp, []byte{0x20, 0x20, 0x20, 0x20, 0x20, 0x2C}, []byte{0x2C}, -1)
		tmp = bytes.Replace(tmp, []byte{0x20, 0x20, 0x20, 0x20, 0x2C}, []byte{0x2C}, -1)
		tmp = bytes.Replace(tmp, []byte{0x20, 0x20, 0x20, 0x2C}, []byte{0x2C}, -1)
		tmp = bytes.Replace(tmp, []byte{0x20, 0x20, 0x2C}, []byte{0x2C}, -1)
		tmp = bytes.Replace(tmp, []byte{0x20, 0x2C}, []byte{0x2C}, -1)

		//For black line between TX and RX lines
		//RX this line and TX last line
		if bytes.HasPrefix(tmp, []byte{82, 88, 44}) {
			if lastIsTX {
				out = append(out, []byte{})
			} else if i != 0 && !bytes.Equal(tmp, out[len(out)-1]) {
				out = append(out, []byte{})
			}
		}
		//TX this line
		if bytes.HasPrefix(tmp, []byte{84, 88, 44}) {
			lastIsTX = true
		} else {
			lastIsTX = false
		}
		out = append(out, tmp)
	}
	return &out
}

//TX,663,06,F1 04 71 01 0F 0C,
func lengthError(lines *[][]byte) {
	for i, line := range *lines {
		if len(line) > 7 && line[7] == byte(0x30) {
			(*lines)[i] = append(line[:7], line[8:]...)
		}
	}
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
