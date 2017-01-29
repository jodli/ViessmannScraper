package main

import (
  "fmt"
  "net"
  "bufio"
  "os"
)

func Connect() (bool, net.Conn) {
  fmt.Println("Connecting...")
  conn, err := net.Dial("tcp", "raspberrypi-2:3002")
  
  if err != nil {
    fmt.Println(err)
    return false, nil
  }
  fmt.Print("Connected to: ")
  fmt.Println(conn.RemoteAddr())
  return true, conn
}

func main() {
  fmt.Println("ViessmannScraper")
  
  connected, conn := Connect()
  if !connected {
    os.Exit(1)
  }

  fmt.Println("Writing to server...")
  fmt.Fprintf(conn, "device\r\n")

  connbuf := bufio.NewReader(conn)
  for {
    fmt.Println("Reading...")
    str, err := connbuf.ReadString('\n')

    if len(str) > 0 {
      fmt.Printf("Received %d bytes: %s", len(str), str)
    }
    if err != nil {
      break
    }
  }
}
