package main

import (
  "fmt"
  "net"
  "bufio"
  "os"
  "flag"
)

func Connect(address string, port int) (bool, net.Conn) {
  addressStr := fmt.Sprintf("%s:%d", address, port)
  fmt.Println("Connecting to:", addressStr)

  conn, err := net.Dial("tcp", addressStr)
  
  if err != nil {
    fmt.Println(err)
    return false, nil
  }

  fmt.Println("Connected to:", conn.RemoteAddr())
  return true, conn
}

func main() {
  fmt.Println("ViessmannScraper")
  
  addressPtr := flag.String("address", "raspberrypi-2", "The address of the vcontrold telnet server.")
  portPtr := flag.Int("port", 3002, "The port of the vcontrold telnet server.")

  flag.Parse()

  connected, conn := Connect(*addressPtr, *portPtr)
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
