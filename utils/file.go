package utils

import (
	"bufio"
	"os"
)

/*
    brief:获取文件行数
*/
func GetFileLineCount(filePath string) int{
	file,err := os.Open(filePath)
	if err!=nil{
		return 0
	}
	fileScanner := bufio.NewScanner(file)
	lineCount := 0
	for fileScanner.Scan() {
		lineCount++
	}
	return lineCount
}

/*
    brief:判定文件是否存在
*/
func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}