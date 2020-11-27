package main

import (
    "fmt"
    "net"
    "os"
    "bufio"
    "encoding/gob"
    "osproject.qu/dhcp"
)

const (
    discover = "DHCPDISCOVER"
    request = "DHCPREQUEST"
)

func main() {
    // Send a connection request to the server
    conn, err := net.Dial("tcp", "localhost:8000")

    if err != nil{
        fmt.Println("Error Connection", err)
        os.Exit(1)
    }

    defer conn.Close()

    // printing the ip and port
    fmt.Println("Successfully Connected with", conn.RemoteAddr().(*net.TCPAddr).IP, "on port", conn.RemoteAddr().(*net.TCPAddr).Port)

    // Sending the discover message to the server
    fmt.Fprintf(conn, discover + "\n")

    // Getting Server Proposal
    dec:= gob.NewDecoder(conn)
    msg1 := &dhcp.DHCP{}
    dec.Decode(msg1)
    fmt.Printf("\nServer offered \n%+v\n\nI think I like it\n\n", msg1)

    // Sending DHCP request
    fmt.Fprintf(conn, request + "\n")

    // Reading the ack message
    buffer := bufio.NewReader(conn)
    read,_ := buffer.ReadBytes('\n')
    msg2 := string(read[:len(read)-1])
    fmt.Println("The server likes it too\n" + msg2)

    // Handling lease questions
    done := make(chan string)
    go handleLease(done, buffer, conn)

    // Sending and Receiving UDP DNS requests
    dns()
    dns2()

    <-done
}

func handleLease(done chan<- string, buffer *bufio.Reader, conn net.Conn){
    for {
        read, err := buffer.ReadBytes('\n')
        if read != nil && err == nil{
            fmt.Println("\nServer is checking on me")
            fmt.Fprintf(conn, "present\n")
        } else { fmt.Println("Error Reading",err); done<- "Done" }
    }
}

// Simple DNS test
func dns(){
    fmt.Println("\nRequesting ip address of PC 1")

    // Setting up the destination
    conn, _ := net.Dial("udp", ":8001")
    // Sending the domain
    fmt.Fprintf(conn, "PC 1"+"\n")
    // Receiving the Reply
    buf,_,_ := bufio.NewReader(conn).ReadLine()
    fmt.Println(string(buf))
}

// False domain DNS test
func dns2(){
    fmt.Println("\nRequesting ip address of PC 2")

    // Setting up the destination
    conn, _ := net.Dial("udp", ":8001")
    // Sending the domain
    fmt.Fprintf(conn, "PC 2"+"\n")
    // Receiving the Reply
    buf,_,_ := bufio.NewReader(conn).ReadLine()
    fmt.Println(string(buf))
}
