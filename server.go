package main

import (
    "fmt"// strings
    "net" //sockets
    "bufio" //buffers
    "math/rand" //rand
    "time"// time
    "errors"//errors
    "sync"//syncronization
    "encoding/gob"//sending structs over tcp
    "osproject.qu/dhcp"
)

const (
    ack = "DHCPACK"
)

var ip_table = make(map[string]string)
var mutex = &sync.Mutex{}

func main(){
    ip_table["192.168.0.1"] = "Default Gateway"
    ip_table["192.168.0.2"] = "DNS 1"
    ip_table["192.168.0.3"] = "DNS 2"
    ip_table["192.168.255.254"] = "Broadcast"

    // Channel that we will use
    done := make(chan string, 2)

    go tcp_listen(done)
    go udp_listen(done)
    <-done
    <-done
}

func tcp_listen(done chan<- string){
    // Creating a Listiner
    in, err := net.Listen("tcp", ":8000")

    if err != nil {
        fmt.Println("Error Listening:", err)
        done <- "done"
    }
    defer in.Close()

    fmt.Println("Listener Started")

    // Infinite loop to accept new clients and handle them concurrently
    for {
        conn, err := in.Accept()

        if err != nil{
            fmt.Println("Error Connecting:", err)
            continue
        }

        // Handling each client in a separate thread
        offer_ip := make(chan string, 1)
        go handleConnection(conn, offer_ip)

        fmt.Println("New Connection from IP address", conn.RemoteAddr().(*net.TCPAddr).IP, "came\n")
    }

    done <- "done"
}

func handleConnection(conn net.Conn, offer_ip chan string){
    // Wrapping the connection with buferio reader and reading till the terminator
    buffer, err := bufio.NewReader(conn).ReadBytes('\n')

    if err != nil{
        fmt.Println("The Client Left\n")
        conn.Close()
        return
    }

    // type cast the buffer to a string
    msg := string(buffer[:len(buffer)-1])
    fmt.Println("The client Sent This Message", msg, "\n")

    if msg == "DHCPDISCOVER"{
        dhcp_offer, err := makeOffer()
        if err != nil {
            fmt.Println("Error while adding:", err)
        }

        // Sending the proposed ip to the client
        encoder := gob.NewEncoder(conn)
        encoder.Encode(dhcp_offer)
        fmt.Printf("Here is the offer sent to the client:\n%+v\n\n", dhcp_offer)

        // sending the ip to else if
        offer_ip <- dhcp_offer.Ip_offer

        // Get the client reply
        handleConnection(conn, offer_ip)
    } else if msg == "DHCPREQUEST" {
        // receiving the ip from the channel
        ip := <-offer_ip
        fmt.Printf("Client wants the ip \"%s\"\n\n", ip)

        mutex.Lock()
        addPC(ip)
        mutex.Unlock()

        // Sending the Acknowledgment to the client
        fmt.Fprintf(conn, ack + "\n")

        checkLease(conn, ip)
    }
}

// An offer that the client can't refuse ;)
func makeOffer() (*dhcp.DHCP, error) {
    dhcp_offer := dhcp.DHCP{
        Default_gateway : "192.168.0.1",
        Subnet_mask : "255.255.0.0",
        Dns_address : "192.168.0.2",
    }

    // If the number of pcs hasn't reached the limit
    if len(ip_table) < (2<<15 - 1) {
        // Generate a randomly seeded number
        random := rand.NewSource(time.Now().UnixNano())
        // Cannot narrow down the range since ips are generated randomly
        // max := rand.New(random).Intn( (2<<15 - 2) - len(ip_table)) + len(ip_table)
        max := rand.New(random).Intn( 2<<15 - 2 )

        // Formulating the ip address with the magic of math
        fourth_octet := max%256
        third_octet := (max/256) - (fourth_octet/256)
        ip := fmt.Sprintf("192.168.%d.%d", third_octet, fourth_octet)

        // If the ip is not already in the table
        if _, found := ip_table[ip]; !found{
            // Adding the ip to the created struct
            dhcp_offer.Ip_offer = ip
            return &dhcp_offer, nil
        } else {
            // Calling the function again until it gets an open ip
            return makeOffer()
        }
    } else {return &dhcp_offer, errors.New("The matrix is full")}
}

// Add the pc to the table
func addPC(ip string) {
    pc := fmt.Sprintf("PC %d", len(ip_table)-3)
    ip_table[ip] = pc
    fmt.Println(ip_table[ip] + " has been added to the DHCP table with ip " + ip)

    fmt.Println("\nHere is the full table")
    for k,v := range ip_table {
        fmt.Printf("%-20s%-20s\n", k, v)
    }
}

// Sleeping for a day and then chekcing if the client is still alive
func checkLease(conn net.Conn, ip string){
    for{
        // Everyday
        // time.Sleep(24 * time.Hour)
        time.Sleep(5 * time.Second)
        fmt.Println("\nChecking on", ip_table[ip])

        // Send this Message
        fmt.Fprintf(conn, "Check" + "\n")

        // If the bufio function returned an error then the client left
        _, err := bufio.NewReader(conn).ReadBytes('\n')
        if err != nil{
            fmt.Println("The Client Left\n")

            mutex.Lock()
            // Remove the entry
            delete(ip_table, ip)
            mutex.Unlock()

            return
        } else { fmt.Println("He is there") }
    }
}

// Listening to client UDP requests and replying to them
func udp_listen(done chan<- string){
    // Creating a Listiner
    conn, err := net.ListenPacket("udp", ":8001")

    if err != nil {
        fmt.Println("Error Listening:", err)
        done <- "done"
    }

    go dns_reply(conn)
}

// Simple DNS implementation
func dns_reply(conn net.PacketConn){
    for {
        buf := make([]byte, 1024)
        length, source, _ := conn.ReadFrom(buf)
        fmt.Println("\nThis source sent a UDP message:", source)
        fmt.Println("Here is the message:",string(buf[:length]))

        // Checking if the domain is in the map
        found := false
        for k,v := range ip_table {
            if v == string(buf[:length-1]) {
                buf = []uint8("The IP address is " + k + "\n")
                conn.WriteTo(buf, source)
                found = true
            }
        }

        // If the "Domain" is not found
        if !found{
            buf = []uint8("The requested domain name is currently not available" + "\n")
            conn.WriteTo(buf, source)
        }
    }
}
