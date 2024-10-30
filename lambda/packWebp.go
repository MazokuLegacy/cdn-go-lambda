package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"
)

func packWebp(inputs [][]byte, _ int) ([]byte, error) {
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
		"-resize", "400x",
		outPath)

	cmdstrt := time.Now()
	err = cmd.Start()
	if err != nil {
		fmt.Println("Error starting command:", err)
		return nil, err
	}

	err = cmd.Wait()
	fmt.Println("cmd ran in ", time.Since(cmdstrt))
	if err != nil {
		fmt.Println("Error waiting for command:", err)
		return nil, err
	}

	output, err := io.ReadAll(outFile)
	return output, nil
}
