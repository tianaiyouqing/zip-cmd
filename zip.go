package main

import (
	"archive/zip"
	"flag"
	"fmt"
	"github.com/sabhiram/go-gitignore"
	"github.com/schollz/progressbar/v3"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	ignoreFlag := flag.String("ignore", "", "额外的忽略规则（使用逗号分隔），例如：\"*.log,tmp/,secret.txt\"")
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		fmt.Println("用法: zip [--ignore=规则] <源文件夹> [输出zip路径] \n" +
			"=====================================\n" +
			"--ignore 选填， 可配置.zipignore文件一起使用\n" +
			"<源文件夹> 必填, 要压缩的文件路径\n" +
			"<输出zip路径> 选填, 默认使用文件夹名称\n" +
			"=====================================\n" +
			"例子1(对xxx文件压缩): zip ../xxx \n" +
			"例子2(对xxx文件压缩,指定输出目录): zip ../xxx  ../xxx.zip\n" +
			"例子3(对xxx文件压缩,指定输出目录,忽略target和dist目录): zip --ignore=\"target,dist\" ../xxx  ../xxx.zip")
		return
	}

	srcDir := args[0]
	var outputZip string
	if len(args) >= 2 {
		outputZip = args[1]
	} else {
		folderName := filepath.Base(filepath.Clean(srcDir))
		outputZip = folderName + ".zip"
		fmt.Println("未指定输出文件名，使用默认:", outputZip)
	}

	// 组合忽略规则
	var patterns []string
	ignorePath := filepath.Join(srcDir, ".zipignore")
	if _, err := os.Stat(ignorePath); err == nil {
		lines, _ := os.ReadFile(ignorePath)
		fmt.Println("使用 .zipignore 忽略规则")
		for _, line := range strings.Split(string(lines), "\n") {
			if strings.TrimSpace(line) != "" {
				patterns = append(patterns, line)
			}
		}
	}

	// 添加命令行指定的规则
	if *ignoreFlag != "" {
		cmdRules := strings.Split(*ignoreFlag, ",")
		for _, rule := range cmdRules {
			rule = strings.TrimSpace(rule)
			if rule != "" {
				patterns = append(patterns, rule)
			}
		}
	}

	ign := ignore.CompileIgnoreLines(patterns...)

	// 获取所有需要压缩的文件
	var filesToZip []string
	basePath := filepath.Clean(srcDir)

	err := filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(basePath, path)
		if err != nil {
			return err
		}

		if ign.MatchesPath(relPath) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !info.IsDir() {
			filesToZip = append(filesToZip, relPath)
		}
		return nil
	})
	if err != nil {
		fmt.Println("扫描文件失败:", err)
		return
	}

	zipFile, err := os.Create(outputZip)
	if err != nil {
		fmt.Println("无法创建 zip 文件:", err)
		return
	}
	defer zipFile.Close()
	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	bar := progressbar.Default(int64(len(filesToZip)))
	for _, relPath := range filesToZip {
		fullPath := filepath.Join(basePath, relPath)
		file, err := os.Open(fullPath)
		if err != nil {
			fmt.Printf("\n无法打开: %s\n", relPath)
			continue
		}
		writer, err := zipWriter.Create(relPath)
		if err != nil {
			file.Close()
			fmt.Printf("\n写入 zip 条目失败: %s\n", relPath)
			continue
		}
		_, err = io.Copy(writer, file)
		file.Close()
		if err != nil {
			fmt.Printf("\n写入失败: %s\n", relPath)
			continue
		}
		bar.Add(1)
	}

	fmt.Println("\n✅ 压缩完成:", outputZip)
}
