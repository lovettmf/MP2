package main

import (
	"bufio"
	"bytes"
	"encoding/gob"
	"fmt"
	"net"
	"os"
	"strings"
)

type Message struct {
	To      string
	From    string
	Content string
}

func rec(c net.Conn, exit chan int) {
	for {
		// create a temp buffer
		tmp := make([]byte, 500)
		c.Read(tmp)

		// convert bytes into Buffer (which implements io.Reader/io.Writer)
		tmpbuff := bytes.NewBuffer(tmp)
		tmpstruct := new(Message)

		// creates a decoder object
		gobobjdec := gob.NewDecoder(tmpbuff)
		// decodes buffer and unmarshals it into a Message struct
		gobobjdec.Decode(tmpstruct)
		fmt.Println(tmpstruct.Content)

		if tmpstruct.Content == "exit" {
			exit <- 1
			return
		}
	}
}

func send(c net.Conn, exit chan int, username string, freshInput chan int) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print(">> ")
	text, _ := reader.ReadString('\n')

	input := strings.Split(text, " ")

	if strings.ToLower(input[0]) == "send" {
		freshInput <- 1

		to := input[1]                          //username of recipient
		content := strings.Join(input[2:], " ") //message content

		message := Message{To: to, From: username, Content: content}

		bin_buf := new(bytes.Buffer)

		// create a encoder object
		gobobj := gob.NewEncoder(bin_buf)
		// encode buffer and marshal it into a gob object
		gobobj.Encode(message)

		c.Write(bin_buf.Bytes())

	} else if strings.TrimSpace(string(text)) == "EXIT" { //Check for exit command
		fmt.Println("TCP client exiting...")
		exit <- 1
		return
	}
	freshInput <- 1
}

func main() {
	arguments := os.Args
	if len(arguments) < 4 {
		fmt.Println("Please provide host, port, and username.")
		return
	}

	username := arguments[3]

	CONNECT := arguments[1] + ":" + arguments[2]
	c, err := net.Dial("tcp", CONNECT)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Fprintf(c, username+"\n")

	exit := make(chan int, 1)
	freshInput := make(chan int, 1)

	go rec(c, exit)

	for {
		go send(c, exit, username, freshInput)

		select {
		case <-freshInput:
			continue
		case <-exit:
			fmt.Println("connection closed")
			exit <- 1
			return
		}

	}

	//ok maybe only have one goroutine, but which one?
	//need to add selects so that they all exit

}
