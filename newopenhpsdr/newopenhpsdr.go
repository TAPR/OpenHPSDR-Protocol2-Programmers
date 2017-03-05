// Package to interface with the openHPSDR Radio Boards
// Using the new protocol
// by Dave Larsen KV0S, May 3, 2014
// GPL2
//
package newopenhpsdr

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type Intface struct {
	Intname   string `json:"intname"`
	Matchname string `json:"matchname"`
	Index     int    `json:"index"`
	MAC       string `json:"mac"`
	Ipv4      string `json:"ipv4"`
	Ipv6      string `json:"ipv6"`
	Ipv4Bcast string `json:"ipv4bcast"`
}

type Hpsdrboard struct {
	Status     string `json:"status"`
	Board      string `json:"board"`
	Baddress   string `json:"baddress"`
	Atlas      Atlasboards
	Pcaddress  string `json:"pcaddress"`
	Firmware   string `json:"firmware"`
	Protocol   string `json:"protocol"`
	Clock      int32  `json:"clock"`
	Receivers  int    `json:"receivers"`
	Freqinput  string `json:"freqinput"`
	Iqdata string `json:"iqdata"`
	Mac        []byte `json:"mac"`
	Macaddress string `json:"macaddress"`
}

type Atlasboards struct {
	Mercury1 string `json:"mercury1"`
	Mercury2 string `json:"mercury2"`
	Mercury3 string `json:"mercury3"`
	Mercury4 string `json:"mercury4"`
	Penelope string `json:"penelope"`
	Metis    string `json:"metis"`
}

type SetIPmessage struct {
	Oldaddress string `json:"oldadress"`
	Newaddress string `json:"newadress"`
	Macaddress string `json:"macaddress"`
	Message    string `json:"message"`
}

type Erasemessage struct {
	Ltime   chan int
	Message string `json:"message"`
	Done    chan bool
}

type Packetstats struct {
	Filename   string `json:"filename"`
	Filesize   string `json:"filesize"`
	Memorysize string `json:"memorysize"`
	Packets    string `json:"packets"`
}

type Programmessage struct {
	Packetpercent string `json:"packetpercent"`
	Message       string `json:"message"`
}

//  format Intface struct for web output
func Intfacetable(intf Intface) (str string) {
	str = fmt.Sprintf("<tr><td align=\"right\"><b>%d:</b></td><td> %s (%s) (%s) (%s)</td></tr>\n", intf.Index, intf.Intname, intf.MAC, intf.Ipv4, intf.Ipv6)
	return str
}

//  format Hpsdrboard struct for web output
func Hpsdrboardtable(brd Hpsdrboard) (str string) {
	var strs []string
	s := fmt.Sprintf("<tr><td align=\"right\"><b>Board:</b>  </td><td> %s</td></tr>\n", brd.Board)
	strs = append(strs, s)
	s = fmt.Sprintf("<tr><td align=\"right\"><b>Board Mac:</b>  </td><td> %s</td></tr>\n", brd.Macaddress)
	strs = append(strs, s)
	s = fmt.Sprintf("<tr><td align=\"right\"><b>Board Address:</b>  </td><td> %s</td></tr>\n", brd.Baddress)
	strs = append(strs, s)
	s = fmt.Sprintf("<tr><td align=\"right\"><b>Board Status:</b>  </td><td> %s</td></tr>\n", brd.Status)
	strs = append(strs, s)
	s = fmt.Sprintf("<tr><td align=\"right\"><b>Protocol:</b>  </td><td> %s</td></tr>\n", brd.Protocol)
	strs = append(strs, s)
	s = fmt.Sprintf("<tr><td align=\"right\"><b>Firmware:</b>  </td><td> %s</td></tr>\n", brd.Firmware)
	strs = append(strs, s)
	s = fmt.Sprintf("<tr><td align=\"right\"><b>Receivers:</b>  </td><td> %d</td></tr>\n", brd.Receivers)
	strs = append(strs, s)
	s = fmt.Sprintf("<tr><td align=\"right\"><b>Frequency Input:</b>  </td><td> %s</td></tr>\n", brd.Freqinput)
	strs = append(strs, s)
	s = fmt.Sprintf("<tr><td align=\"right\"><b>IQ data format:</b>  </td><td> %s</td></tr>\n", brd.Iqdata)
	strs = append(strs, s)
	str = strings.Join(strs, "")
	return str
}

//  format Hpsdrboard struct for web output
func Hpsdrboardlist(brd Hpsdrboard) (str string) {
	str = fmt.Sprintf("<tr><td align=\"right\"><b>%s:</b>  </td><td> (%s) (%s)</td></tr>\n", brd.Board, brd.Macaddress, brd.Baddress)
	return str
}

//  format Hpsdrboard struct for web output
func Atlasboardstable(brd Hpsdrboard) (str string) {
	str = fmt.Sprintf("%#v\n<br/>", brd)
	return str
}

// Reset the Hpsdrboard to no value
func ResetHpsdrboard(str Hpsdrboard) Hpsdrboard {
	str.Status = ""
	str.Board = ""
	str.Baddress = ""
	str.Atlas.Mercury1 = ""
	str.Atlas.Mercury2 = ""
	str.Atlas.Mercury3 = ""
	str.Atlas.Mercury4 = ""
	str.Atlas.Penelope = ""
	str.Atlas.Metis = ""
	str.Pcaddress = ""
	str.Firmware = ""
	str.Protocol = ""
	str.Clock = 0
	str.Receivers = 0
	str.Freqinput = ""
	str.Mac = nil
	str.Macaddress = ""
	return str
}

// Determine the network interfaces connect to this machine.
func Interfaces() (Intfc []Intface) {

	intr, err := net.Interfaces()
	if err != nil {
		log.Println("Interface error", err)
	}

	Intfc = make([]Intface, len(intr))

	for i := range intr {
		//log.Println(intr[i].Index, intr[i].Name, intr[i].HardwareAddr)
		Intfc[i].Intname = intr[i].Name
		Intfc[i].Index = intr[i].Index
		Intfc[i].Matchname = strings.Replace(intr[i].Name, " ", "+", -1)
		Intfc[i].MAC = intr[i].HardwareAddr.String()
		aad, err := intr[i].Addrs()
		if err != nil {
			log.Println("Interface error", err)
		}

		for j := range aad {
			//	ip := net.ParseIP(aad[j].String())
			str := aad[j].String()

			if strings.Contains(str, ".") {
				if runtime.GOOS == "windows" {
					//_ = net.ParseIP(aad[j].String())
					//Intfc[i].Ipv4 = aad[j].String()
					//} else if runtime.GOOS == "darwin" {
					//ip := net.ParseIP(aad[j].String())
					//Intfc[i].Ipv4 = ip.String()
					adstring := strings.Split(aad[j].String(), "/")
					Intfc[i].Ipv4 = adstring[0]
				} else {
					ip, _, err := net.ParseCIDR(aad[j].String())
					if err != nil {
						log.Println("Parse CIDR error", err)
					}
					Intfc[i].Ipv4 = ip.String()
				}

				//str := strings.Split(Intfc[i].Ipv4, ".")
				var ipd []string
				//ipd = append(ipd, str[0], str[1], str[2], "255")
				ipd = append(ipd, "255", "255", "255", "255")
				Intfc[i].Ipv4Bcast = strings.Join(ipd, ".")
			} else {
				ip, _, err := net.ParseCIDR(aad[j].String())
				if err != nil {
					log.Println("Parse CIDR error", err)
				}
				Intfc[i].Ipv6 = ip.String()
			}
		}
	}
	return Intfc
}

/*
func Intfacecompare(intr Intface, intf Intface) bool {
	var match bool
	match = false
	if intr.MAC == intf.MAC {
		match = true
	}
	return match
}
*/

func Commlink(addrStr string) (l *net.UDPConn, err error) {
	//log.Println("Commlink:", addrStr)
	addrst := strings.Split(addrStr, ":")
	addr, err := net.ResolveUDPAddr("udp", addrst[0]+":0")
	if err != nil {
		log.Println(" Addr not resolved ", err)
	}

	//log.Println("Commlink:", addrStr)
	l, err = net.ListenUDP("udp", addr)
	if err != nil {
		log.Println(" listenUDP error ", err)
	}
	//log.Println("Commlink:", addrStr)

	//l.SetReadDeadline(time.Time(time.Now().Add(100 * time.Nanosecond)))

	return l, err
}

func Commpacketsend(l *net.UDPConn, destStr string, snd []byte) (k int, err error) {
	dest, err := net.ResolveUDPAddr("udp", destStr)
	if err != nil {
		log.Println(" broadcast not resolved ", err)
	}

	k, err = l.WriteToUDP(snd, dest)
	if err != nil {
		log.Println(" broadcast not connected ", k, err)
	}
	return k, err
}

func Commpacketreceive(l *net.UDPConn) (num int, ad *net.UDPAddr, rec []byte, err error) {
	rec = make([]byte, 60, 60)
	for {
		//log.Println("In receiver before read packet")
		num, ad, err = l.ReadFromUDP(rec)

		if err != nil {
			log.Println(" UDP Read error ", err)
		}
		//log.Printf("%d::%v::%+v\n", num, ad, rec)

		if num > 0 {
			break
		}
	}

	return num, ad, rec, err
}

func Makepacket(packettype string, seq int32, debug string) (buf []byte, err error) {
	buf = make([]byte, 60, 60)
	err = nil
	switch packettype {
	case "discover":
		binary.BigEndian.PutUint32(buf, uint32(seq))
		buf[4] = 0x02

		for i := 5; i < 60; i++ {
			buf = append(buf, 0x00)
		}
		if strings.Contains(debug, "dec") {
			log.Printf(" %d\n", buf)
		} else if strings.Contains(debug, "hex") {
			log.Printf(" %x\n", buf)
		} else {
			//log.Printf(" \n")
		}
		return buf, err

	case "erase":
		binary.BigEndian.PutUint32(buf, uint32(seq))
		buf[4] = 0x04

		for i := 5; i < 60; i++ {
			buf = append(buf, 0x00)
		}
		if strings.Contains(debug, "dec") {
			log.Printf(" %d\n", buf)
		} else if strings.Contains(debug, "hex") {
			log.Printf(" %x\n", buf)
		} else {
			//log.Printf(" \n")
		}
		return buf, err
	case "setip":
		binary.BigEndian.PutUint32(buf, uint32(seq))
		buf[4] = 0x03

		for i := 5; i < 60; i++ {
			buf = append(buf, 0x00)
		}
		if strings.Contains(debug, "dec") {
			log.Printf(" %d\n", buf)
		} else if strings.Contains(debug, "hex") {
			log.Printf(" %x\n", buf)
		} else {
			//log.Printf(" \n")
		}
		return buf, err
	default:
		log.Println("Unknown packettype", packettype)
	}
	return buf, err
}

func Makepacketprogram(ibf []byte, seq uint32, numblk uint32, debug string) (buf []byte, err error) {
	buf = make([]byte, 265, 265)
	err = nil

	binary.BigEndian.PutUint32(buf, seq)
	buf[4] = 0x05
	binary.BigEndian.PutUint32(buf[5:9], numblk)
	for i := 9; i < 265; i++ {
		buf[i] = ibf[i-9]
	}

	if strings.Contains(debug, "dec") {
		log.Printf(" %d\n", buf)
	} else if strings.Contains(debug, "hex") {
		log.Printf(" %x\n", buf)
	} else {
		//log.Printf(" \n")
	}
	return buf, err
}

// Send the Discovery packet to an interface.
func Discover(addrStr string, bcastStr string, debug string) (strs []Hpsdrboard, er error) {
	var b []byte
	var str Hpsdrboard
	str.Mac = make([]byte, 6, 6)

	log.Printf("          Discover: %s -> %s", addrStr, bcastStr)

	b, er1 := Makepacket("discover", 0, debug)
	if er1 != nil {
		log.Println("Error After Makepacket", er1)
	}
	//log.Println("After Makepacket", b)

	l, err := Commlink(addrStr)

	//log.Println("After Commlink", b)
	n, err := Commpacketsend(l, bcastStr, b)
	if err != nil {
		log.Println("Commpacketsend", n, err)
	}

	//log.Println("After Commpacketsend", b)
	n, ad, c, err := Commpacketreceive(l)
	if err != nil {
		log.Println("Commpacketreceive", n, ad, err)
	}

	if strings.Contains(debug, "dec") {
		log.Printf("     Received data: %v bytes from %v   %+v\n", n, ad, c)
	} else if strings.Contains(debug, "hex") {
		log.Printf("     Received data: %v bytes from %v   %x\n", n, ad, c)
	} else {
		log.Printf("     Received data: %v bytes from %v\n", n, ad)
	}

	str.Mac[0] = c[5]
	str.Mac[1] = c[6]
	str.Mac[2] = c[7]
	str.Mac[3] = c[8]
	str.Mac[4] = c[9]
	str.Mac[5] = c[10]

	str.Macaddress = fmt.Sprintf("%x:%x:%x:%x:%x:%x", c[5], c[6], c[7], c[8], c[9], c[10])
	str.Pcaddress = addrStr

	if c[4] == 2 {
		str.Status = "not running"
	} else if c[4] == 3 {
		str.Status = "running"
	}

	if c[11] == 0 {
		str.Board = "ATLAS"
	} else if c[11] == 1 {
		str.Board = "HERMES"
	} else if c[11] == 2 {
		str.Board = "HERMES"
	} else if c[11] == 3 {
		str.Board = "ANGELIA"
	} else if c[11] == 4 {
		str.Board = "ORION"
	} else if c[11] == 4 {
		str.Board = "ANAN-10E"
	} else if c[11] == 6 {
		str.Board = "HERMES-LITE"
	} else {
		str.Board = "Unknown"
	}

	str.Protocol = fmt.Sprintf("%d.%d", c[12]/10, c[12]%10)
	str.Firmware = fmt.Sprintf("%d.%d", c[13]/10, c[13]%10)
	str.Atlas.Mercury1 = fmt.Sprintf("%d.%d", c[14]/10, c[14]%10)
	str.Atlas.Mercury2 = fmt.Sprintf("%d.%d", c[15]/10, c[15]%10)
	str.Atlas.Mercury3 = fmt.Sprintf("%d.%d", c[16]/10, c[16]%10)
	str.Atlas.Mercury4 = fmt.Sprintf("%d.%d", c[17]/10, c[17]%10)
	str.Atlas.Penelope = fmt.Sprintf("%d.%d", c[18]/10, c[18]%10)
	str.Atlas.Metis = fmt.Sprintf("%d.%d", c[19]/10, c[19]%10)

	str.Receivers = int(c[20])
	if c[21] == 0 {
		str.Freqinput = "Frequency"
	} else {
		str.Freqinput = "Phase_word"
	}

	log.Printf("%b", c[22])
	if c[22] == 0 {
		str.Iqdata = "Big-Endian IQ in 3 byte format"
	}else if c[22] == 1 {
		str.Iqdata = "Little-Endian"
	}else if c[22] == 2 {
		str.Iqdata = "3 Byte format"
	}else if c[22] == 3 {
		str.Iqdata = "1 Float format"
	}else if c[22] == 4 {
		str.Iqdata = "1 Double format"
	}

	str.Baddress = ad.String()
	strs = append(strs, str)

	l.Close()

	return strs, nil

}

// Send the Set IP packet to an interface.
func Setip(addrStr string, bcastStr string, str Hpsdrboard, nadr string, debug string) (msg SetIPmessage, er error) {
	var b []byte
	log.Printf("       Set IP sent: %s -> %s\n", addrStr, bcastStr)

	b, er1 := Makepacket("setip", 0, debug)
	if er1 != nil {
		log.Println("Error After Makepacket", er1)
	}

	msg.Newaddress = nadr
	msg.Oldaddress = str.Baddress
	msg.Macaddress = str.Macaddress
	msg.Message = "Setting new IP address"

	// insert MAC address
	b[5] = str.Mac[0]
	b[6] = str.Mac[1]
	b[7] = str.Mac[2]
	b[8] = str.Mac[3]
	b[9] = str.Mac[4]
	b[10] = str.Mac[5]

	// insert new address
	newadr := strings.Split(nadr, ".")
	s, _ := strconv.ParseInt(newadr[0], 10, 32)
	b[11] = byte(s)
	s, _ = strconv.ParseInt(newadr[1], 10, 32)
	b[12] = byte(s)
	s, _ = strconv.ParseInt(newadr[2], 10, 32)
	b[13] = byte(s)
	s, _ = strconv.ParseInt(newadr[3], 10, 32)
	b[14] = byte(s)

	l, err := Commlink(addrStr)

	n, err := Commpacketsend(l, bcastStr, b)
	if err != nil {
		log.Println("Commpacketsend", n, err)
	}

	//n, ad, c, err := Commpacketreceive(l)
	//if err != nil {
	//	log.Println("Commpacketreceive", n, ad, err)
	//}

	//if strings.Contains(debug, "dec") {
	//	log.Printf("     Received data: %v bytes from %v   %+v\n", n, ad, c)
	//} else if strings.Contains(debug, "hex") {
	//	log.Printf("     Received data: %v bytes from %v   %x\n", n, ad, c)
	//} else {
	//	log.Printf("     Received data: %v bytes from %v\n", n, ad)
	//}

	l.Close()

	return msg, nil
}

// Send the Erase packet to an interface.
func Erase(addrStr string, str Hpsdrboard, debug string) (er error) {
	var b []byte
	log.Printf("             Erase: %s -> %s\n", addrStr, str.Baddress)

	b, er1 := Makepacket("erase", 0, debug)
	if er1 != nil {
		log.Println("Error After Makepacket", er1)
	}

	log.Printf("             Erase:  After Makepacket\n")
	l, err := Commlink(addrStr)

	log.Printf("             Erase: After Comlink\n")
	_, err = Commpacketsend(l, str.Baddress, b)
	if err != nil {
		log.Println("Commpacketsend", err)
	}

	log.Printf("             Erase: After Commpacket %+v\n", b)
	for i := 0; i < 2; i++ {
		n, ad, c, err := Commpacketreceive(l)
		if err != nil {
			log.Println("Commpacketreceive", err)
		}

		log.Printf("reply %d %d::%v::%+v\n", i+1, n, ad, c)

		if (c[4] == 3) && (binary.BigEndian.Uint32(c[0:4]) == uint32(0)) {
			if strings.Contains(debug, "dec") {
				log.Printf("     Received data: %v bytes from %v   %+v\n", n, ad, c)
			} else if strings.Contains(debug, "hex") {
				log.Printf("     Received data: %v bytes from %v   %x\n", n, ad, c)
			} else {
				if i == 0 {
					log.Printf("     Erase started: %v bytes from %v\n", n, ad)
				} else {
					log.Printf("    Erase Finished: %v bytes from %v\n", n, ad)
				}
			}
		}
	}
	log.Printf("    Erase complete: \n")

	l.Close()

	return nil
}

// Send the New Erase packet to an interface.
func Erasenew(str Hpsdrboard, debug string) (er error) {
	var emsg chan int
	var idx int
	var lapstm chan Erasemessage
	var lt Erasemessage
	lapstm = make(chan Erasemessage)
	emsg = make(chan int)
	log.Printf("             Erase: %s -> %s\n", str.Pcaddress, str.Baddress)

	er1 := Erasefunc(str, emsg, debug)
	if er1 != nil {
		log.Println("Error After Errorfunc", er1)
	}

	//go Lapstime(&lapstm)

	for {
		select {
		case idx = <-emsg:
			if idx == 0 {
				log.Println(" Erase started. ")
			} else {
				log.Println(" Erase finished. ")
				break
			}
		case lt = <-lapstm:
			ss := fmt.Sprintf("%d Seconds. ", lt.Ltime)
			log.Println(ss)
			//if lt.Done {
			log.Println(" Erase timed out.")
			err := errors.New("Erase timed out")
			return err
			//}
			//default:
		}
	}
	log.Printf("    Erase complete: \n")

	return nil
}

// Send and receive the Erase Packets.
func Erasefunc(str Hpsdrboard, emsg chan int, debug string) (er error) {
	var b []byte
	b, er1 := Makepacket("erase", 0, debug)
	if er1 != nil {
		log.Println("Error After Makepacket", er1)
	}

	l, err := Commlink(str.Pcaddress)

	_, err = Commpacketsend(l, str.Baddress, b)
	if err != nil {
		log.Println("Commpacketsend", err)
	}

	var i int
	for {
		n, ad, c, err := Commpacketreceive(l)
		if err != nil {
			log.Println("Commpacketreceive", err)
		}

		log.Printf("reply %d %d::%v::%+v\n", i+1, n, ad, c)

		if c[4] == 3 && binary.BigEndian.Uint32(c[0:4]) == uint32(0) {
			emsg <- i
			if i > 0 {
				break
			}
		}
	}

	l.Close()

	return nil
}

// lapstime counter using a channel
func Lapstime(lps *Erasemessage) {
	log.Println(" In Lapstime. ")
	var i int
	i = 0
	for {
		select {
		//case <-lt.Done:
		//	break
		default:
			time.Sleep(time.Second)
			i++
			log.Println(i)
			lps.Ltime <- i
		}
	}
}

// Send the Program packet to an interface.
func Program(addrStr string, str Hpsdrboard, input string, debug string) (er error) {
	log.Printf("Program: %s -> %s\n", addrStr, str.Baddress)

	// Open the RBF file
	f, err := os.Open(input)
	if err != nil {
		log.Fatal(err)
	}

	defer func() {
		err := f.Close()
		if err != nil {
			log.Fatal(err)
		}
	}()

	// calculate the Statistics of the RBF file
	fi, err := f.Stat()
	if err != nil {
		log.Println("Could not open the file")
	}

	log.Println("      Programming the HPSDR Board")
	packets := uint32(math.Ceil(float64(fi.Size()) / 256.0))
	log.Println("    Found rbf file:", input)
	log.Println("     Size rbf file:", fi.Size())
	log.Println("Size rbf in memory:", ((fi.Size()+255)/256)*256)
	log.Println("           Packets:", packets)
	log.Println(" ")

	r := bufio.NewReader(f)

	l, err := Commlink(addrStr)

	buf := make([]byte, 256)
	ipk := uint32(0)
	for {
		// read a chunk
		n, err := r.Read(buf)
		if err != nil && err != io.EOF {
			log.Fatal(err)
		}

		// No more data in file
		if n == 0 {
			log.Printf("\n     Program complete: \n\n")
			break
		}

		// Pad out the data to complete the packet
		if n < 256 {
			for i := n; i < 256; i++ {
				buf[i] = 0xFF
			}
			n = 256
		}

		b, er1 := Makepacketprogram(buf, ipk, packets, debug)
		if er1 != nil {
			log.Println("Error After Makepacketprogram", er1)
		}

		for {
			n, err := Commpacketsend(l, str.Baddress, b)
			if err != nil {
				log.Println("Commpacketsend", err)
			}

			n, ad, c, err := Commpacketreceive(l)
			if err != nil {
				log.Println("Commpacketreceive", err)
			}

			recnum := binary.BigEndian.Uint32(c[0:4])
			sennum := ipk
			if (c[4] == 4) && (binary.BigEndian.Uint32(c[0:4]) == ipk) {
				ipk++
				if strings.Contains(debug, "dec") {
					log.Printf("     Received data: %v bytes from %v   %+v\n", n, ad, c)
				} else if strings.Contains(debug, "hex") {
					log.Printf("     Received data: %v bytes from %v   %x\n", n, ad, c)
				} else {
					log.Printf("     Received data: sent %d = rec %d, %v bytes from %v", sennum, recnum, n, ad)
				}
				break
			} else if binary.BigEndian.Uint32(c[0:4]) == ipk {
				log.Printf("     Program complete: \n")
				l.Close()
				return nil
			} else {
				outstr := fmt.Sprintf("     Received data: sent %d = rec %d, %v bytes from %v %+v\n", sennum, recnum, n, ad, c)
				panic(outstr)
			}
		}

	}
	l.Close()
	return nil
}
