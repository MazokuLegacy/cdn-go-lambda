package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"time"
)

func packWebp(inputs [][]byte, width int) ([]byte, error) {
	//preparing inputs
	inputfilestrt := time.Now()
	dir := "/tmp/pack"
	err := os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		return nil, err
	}
	err = processFilesParallel(inputs)
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	fmt.Println("input files prepared in ", time.Since(inputfilestrt))

	//preparing output
	outputfilestrt := time.Now()
	outPath := "/tmp/pack/output.webp"
	outFile, err := os.Create(outPath)
	if err != nil {
		return nil, err
	}
	defer outFile.Close()
	fmt.Println("output files prepared in ", time.Since(outputfilestrt))

	cmd := exec.Command("magick",
		"convert", "+append",
		"/tmp/pack/card*.png",
		"-resize", strconv.FormatFloat(float64(width)/float64(len(inputs)), 'f', 2, 64)+"x",
		outPath)

	cmdstrt := time.Now()
	err = cmd.Run()
	if err != nil {
		fmt.Println("Error running command:", err)
		return nil, err
	}
	fmt.Println("cmd ran in ", time.Since(cmdstrt))
	output, err := io.ReadAll(outFile)
	return output, nil
}
