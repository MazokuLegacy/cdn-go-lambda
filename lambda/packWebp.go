package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strconv"
)

func packWebp(inputs map[string][]byte, width int) ([]byte, error) {
	index := 0
	for _, fileData := range inputs {
		inPath := "/tmp/card" + strconv.Itoa(index) + ".png"
		inFile, err := os.Create(inPath)
		if err != nil {
			return nil, err
		}
		defer inFile.Close()
		defer os.Remove(inPath)
		inFile.Write(fileData)
		index = index + 1
	}
	outPath := "/tmp/output.webp"
	outFile, err := os.Create(outPath)
	if err != nil {
		return nil, err
	}
	defer outFile.Close()
	defer os.Remove(outPath)
	cmd := exec.Command("magick",
		"convert", "-append",
		"/tmp/card*.png",
		"-resize", strconv.Itoa(width)+"x",
		outPath)
	err = cmd.Start()
	if err != nil {
		fmt.Println("Error starting command:", err)
		return nil, err
	}
	err = cmd.Wait()
	if err != nil {
		fmt.Println("Error waiting for command:", err)
		return nil, err
	}
	log.Println("completed")
	output, err := io.ReadAll(outFile)
	return output, nil
}
