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

var ignoreFlag string

func init() {
	flag.StringVar(&ignoreFlag, "ignore", "", "额外的忽略规则（使用逗号分隔），例如：\"*.log,tmp/,secret.txt\"")
	flag.Parse()
}

func main() {
	args := flag.Args()
	if len(args) < 1 {
		printUsage()
		return
	}
	srcDir, outputZip := processArgs(args)
	// 加载忽略规则
	ign := loadIgnore(srcDir, ignoreFlag)

	// 获取所有需要压缩的文件
	basePath := filepath.Clean(srcDir)
	filesToZip, err := collectFiles(basePath, ign)
	if err != nil {
		fmt.Println("扫描文件失败:", err)
		return
	}
	if err := createZipArchive(filesToZip, basePath, outputZip); err != nil {
		fmt.Println("\n压缩失败:", err)
		return
	}
	fmt.Println("\n✅ 压缩完成:", outputZip)
}

func createZipArchive(filesToZip []string, basePath string, outputZip string) error {
	bar := progressbar.Default(int64(len(filesToZip)))

	zipFile, err := os.Create(outputZip)
	if err != nil {
		return fmt.Errorf("无法创建zip文件: %w", err)
	}
	defer zipFile.Close()
	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	for _, relPath := range filesToZip {
		fullPath := filepath.Join(basePath, relPath)
		file, err := os.Open(fullPath)
		if err != nil {
			fmt.Printf("\n无法打开: %s\n%s\n", relPath, err)
			continue
		}
		writer, err := zipWriter.Create(relPath)
		if err != nil {
			file.Close()
			fmt.Printf("\n写入 zip 条目失败: %s\n%s\n", relPath, err)
			continue
		}
		_, err = io.Copy(writer, file)
		file.Close()
		if err != nil {
			fmt.Printf("\n写入失败: %s\n%s\n", relPath, err)
			continue
		}
		bar.Add(1)
	}
	return nil
}

func collectFiles(basePath string, ign *ignore.GitIgnore) ([]string, error) {
	var filesToZip []string
	err := filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(basePath, path)
		if err != nil {
			return err
		}

		if ign != nil && ign.MatchesPath(relPath) {
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
	return filesToZip, err

}

func loadIgnore(srcDir string, ignoreFlag string) *ignore.GitIgnore {
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
	if ignoreFlag != "" {
		cmdRules := strings.Split(ignoreFlag, ",")
		for _, rule := range cmdRules {
			rule = strings.TrimSpace(rule)
			if rule != "" {
				patterns = append(patterns, rule)
			}
		}
	}
	if len(patterns) > 0 {
		return ignore.CompileIgnoreLines(patterns...)
	}
	return nil
}

func processArgs(args []string) (string, string) {
	srcDir := args[0]
	if len(args) >= 2 {
		return srcDir, args[1]
	}
	folderName := filepath.Base(filepath.Clean(srcDir))
	outputZip := folderName + ".zip"
	fmt.Println("未指定输出文件名，使用默认:", outputZip)
	return srcDir, outputZip
}

func printUsage() {
	fmt.Println(`用法: zip [--ignore=规则] <源文件夹> [输出zip路径]
=====================================
--ignore 选填，可配置.zipignore文件一起使用
<源文件夹> 必填, 要压缩的文件路径
<输出zip路径> 选填, 默认使用文件夹名称
=====================================
例子1: zip ../xxx
例子2: zip ../xxx ../xxx.zip
例子3: zip --ignore="target,dist" ../xxx ../xxx.zip`)
}
