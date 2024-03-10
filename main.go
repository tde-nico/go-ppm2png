package main

import (
	"bufio"
	"flag"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

const fileHeader = "P3"

func parseHeader(fileScanner *bufio.Scanner, src string) (int, int, int, error) {
	var (
		err    error
		values []string
		width  int
		height int
		maxVal int
	)

	if fileScanner.Scan() && fileScanner.Text() != fileHeader {
		fmt.Printf("Invalid file header: %v\n", src)
		return 0, 0, 0, fmt.Errorf("invalid file header")
	}

	if fileScanner.Scan() {
		line := fileScanner.Text()
		values = strings.Split(line, " ")
		if len(values) != 2 {
			fmt.Printf("Failed to read %s bad dimensions\n", src)
			return 0, 0, 0, err
		}
		if width, err = strconv.Atoi(values[0]); err != nil {
			fmt.Printf("Failed to read %s bad width\n", src)
			return 0, 0, 0, err
		}
		if height, err = strconv.Atoi(values[1]); err != nil {
			fmt.Printf("Failed to read %s bad height\n", src)
			return 0, 0, 0, err
		}
	}

	if fileScanner.Scan() {
		line := fileScanner.Text()
		if maxVal, err = strconv.Atoi(line); err != nil {
			fmt.Printf("Failed to read %s bad max value\n", src)
			return 0, 0, 0, err
		}
	}
	return width, height, maxVal, nil
}

func parseColor(color string, max int) (byte, error) {
	var err error
	var value int

	if value, err = strconv.Atoi(color); err != nil {
		return 0, err
	}
	value = (value * 255) / max
	return byte(value), nil
}

func parseColors(fileScanner *bufio.Scanner, src string, imageData *image.RGBA, m int) error {
	var err error
	var row int = 0

	for fileScanner.Scan() {
		line := fileScanner.Text()
		values := strings.Split(line, " ")
		if len(values) != 3 {
			fmt.Printf("Invalid line in %v: %v\n", src, line)
			return err
		}

		imageData.Pix[row], err = parseColor(values[0], m)
		if err != nil {
			fmt.Printf("Invalid color in %v at line %v: %v\n", src, line, values[0])
			return err
		}
		imageData.Pix[row+1], err = parseColor(values[1], m)
		if err != nil {
			fmt.Printf("Invalid color in %v at line %v: %v\n", src, line, values[0])
			return err
		}
		imageData.Pix[row+2], err = parseColor(values[2], m)
		if err != nil {
			fmt.Printf("Invalid color in %v at line %v: %v\n", src, line, values[0])
			return err
		}
		imageData.Pix[row+3] = 255

		row += 4
	}

	return nil
}

func convert(src, dst string, wg *sync.WaitGroup) {
	if wg != nil {
		defer wg.Done()
	}
	fmt.Printf("Converting %s to %s\n", src, dst)

	readFile, err := os.Open(src)
	if err != nil {
		fmt.Printf("Could not open file: %v\n", err)
		return
	}
	defer readFile.Close()
	fileScanner := bufio.NewScanner(readFile)
	fileScanner.Split(bufio.ScanLines)

	w, h, m, err := parseHeader(fileScanner, src)
	if err != nil {
		return
	}

	imageData := image.NewRGBA(image.Rect(0, 0, w, h))
	parseColors(fileScanner, src, imageData, m)
	if err != nil {
		return
	}

	writeFile, err := os.Create(dst)
	if err != nil {
		fmt.Printf("Could not create file: %v\n", err)
		return
	}
	defer writeFile.Close()

	png.Encode(writeFile, imageData)
	if wg != nil {
		fmt.Printf("Converted %s to %s\n", src, dst)
	}
}

func main() {
	var (
		inputDir  string
		outputDir string
		parallel  bool
		h         bool
		help      bool
	)

	flag.StringVar(&inputDir, "i", "", "Input directory")
	flag.StringVar(&outputDir, "o", "", "Output directory")
	flag.BoolVar(&parallel, "p", false, "Run in parallel")
	flag.BoolVar(&h, "h", false, "Print help")
	flag.BoolVar(&help, "help", false, "Print help")
	flag.Parse()

	if h || help {
		flag.PrintDefaults()
		return
	}
	if inputDir == "" || outputDir == "" {
		flag.Usage()
		os.Exit(1)
	}

	entries, err := os.ReadDir(inputDir)
	if err != nil {
		fmt.Printf("Could not read directory: %v\n", err)
		os.Exit(1)
	}

	var wg sync.WaitGroup
	for _, file := range entries {
		if file.IsDir() {
			continue
		}
		input := filepath.Join(inputDir, file.Name())
		output := filepath.Join(outputDir, strings.TrimSuffix(file.Name(), ".ppm")+".png")
		if parallel {
			wg.Add(1)
			go convert(input, output, &wg)
		} else {
			convert(input, output, nil)
		}
	}
	if parallel {
		wg.Wait()
	}
}
