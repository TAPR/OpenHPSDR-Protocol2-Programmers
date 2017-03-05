// Program to program HPSDR boards from the command line
// new protocol version

// by David R. Larsen KV0S, Copyright 2014-11-24
//
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"runtime"
	"strings"
	"time"

	"oak.snr.missouri.edu/daveradio/newopenhpsdr"
)

const version string = "0.2.8"
const protocol string = ">1.7"
const update string = "2016-9-17"

//  global current board
var crtbd newopenhpsdr.Hpsdrboard

// function to point users to the command list
func usage() {
	log.Printf("    For a list of commands use -help \n\n")
}

// Function to print the program name info
func program() {
	log.Printf("HPSDRProgrammer_cmd  version:(%s)\n", version)
	log.Printf("    By Dave KV0S, 2014-11-24, GPL2 \n\n")
	log.Printf("        Protocol: %s \n", protocol)
	log.Printf("    Last Updated: %s \n\n", update)
}

// Convenience function to print board data
func Listboard(str newopenhpsdr.Hpsdrboard) {
	if str.Macaddress != "0:0:0:0:0:0" {
		log.Printf("\n")
		log.Printf("        Board Type: %s\n", str.Board)
		log.Printf("       HPSDR Board: (%s)\n", str.Macaddress)
		log.Printf("     Board Address: %s\n", str.Baddress)
		log.Printf("          Protocol: %s\n", str.Protocol)
		log.Printf("          Firmware: %s\n", str.Firmware)
		log.Printf("         Receivers: %d\n", str.Receivers)
		log.Printf("       Freq. Input: %s\n", str.Freqinput)
		log.Printf("    IQ data format: %s\n", str.Iqdata)
		log.Printf("            Status: %s\n", str.Status)
	}
}

// Convenience function to print interface data
func Listinterface(itr newopenhpsdr.Intface) {
	log.Printf("          Computer: (%v)\n", itr.MAC)
	log.Printf("                OS: %s (%s) %d CPU(s)\n", runtime.GOOS, runtime.GOARCH, runtime.NumCPU())
	if runtime.GOARCH != "arm" {
		u, err := user.Current()
		if err != nil {
			panic(err.Error())
		}
		log.Printf("          Username: %s (%s) %s\n", u.Name, u.Username, u.HomeDir)
	}
	log.Printf("              IPV4: %v\n", itr.Ipv4)
	//log.Printf("              Mask: %d\n", itr.Mask)
	//log.Printf("           Network: %v\n", itr.Network)
	log.Printf("              IPV6: %v\n", itr.Ipv6)
}

func Listflags(fg flagsettings) {
	log.Printf("    Saved Settings: \n")
	log.Printf("         Interface: %v\n", fg.Intface)
	log.Printf("             Index: %v\n", fg.Index)
	log.Printf("          Filename: %v\n", fg.Filename)
	log.Printf("      Selected MAC: (%v)\n", fg.SelectMAC)
	log.Printf("            SetRBF: %v\n", fg.SetRBF)
	log.Printf("             Debug: %v\n", fg.Debug)
	log.Printf("            Ddelay: %d\n", fg.Ddelay)
	log.Printf("            Edelay: %d\n", fg.Edelay)
}

func Listflagstemp(fgt flagtemp) {
	log.Printf("     Temp settings: \n")
	log.Printf("          Settings: %v\n", fgt.Settings)
	log.Printf("             SetIP: %v\n", fgt.SetIP)
	log.Printf("              Save: %v\n", fgt.Save)
	log.Printf("              Load: %v\n", fgt.Load)
}

func Initflags(fg *flagsettings) {
	fg.Intface = "none"
	fg.Filename = "none"
	fg.SelectMAC = "none"
	fg.SetRBF = "none"
	fg.Debug = "none"
	fg.Ddelay = 2
	fg.Edelay = 60
}

type flagsettings struct {
	Filename  string
	Intface   string
	Index     int
	SelectMAC string
	SetRBF    string
	Debug     string
	Ddelay    int
	Edelay    int
}

type flagtemp struct {
	SetIP    string
	Settings string
	Save     string
	Load     string
}

func Initflagstemp(fgt *flagtemp) {
	fgt.SetIP = "none"
	fgt.Settings = "none"
	fgt.Save = "none"
	fgt.Load = "none"
}

func Parseflagstruct(fg *flagsettings, fgt *flagtemp, id int, stmac string, stip string, strbf string, db string, ss string, sv string, ld string, dd int, ed int) {

	Initflags(fg)
	Initflagstemp(fgt)

	if (ld == "default") || (ld == "Default") {
		fg.Filename = "HPSDRProgrammer_cmd.json"
	} else if ld != "none" {
		fg.Filename = ld
	}

	if ld != "none" {

		dta, _ := ioutil.ReadFile(fg.Filename)
		err := json.Unmarshal(dta, &fg)
		if err != nil {
			log.Println("error:", err)
		}
	}

	//if ifn != "none" {
	//	fg.Intface = ifn
	//}
	if id != 0 {
		fg.Index = id
	}
	if stmac != "none" {
		fg.SelectMAC = stmac
	}
	if strbf != "none" {
		fg.SetRBF = strbf
	}
	if db != "none" {
		fg.Debug = db
	}
	if ed != 20 {
		fg.Edelay = ed
	}
	if dd != 2 {
		fg.Ddelay = dd
	}
	if ed != 2 {
		fg.Edelay = ed
	}
	if stip != "none" {
		fgt.SetIP = stip
	}
	if ss != "none" {
		fgt.Settings = ss
	}
	if sv == "default" {
		fgt.Save = sv
		fg.Filename = "HPSDRProgrammer_cmd.json"
	} else if sv != "none" {
		fg.Filename = sv
		fgt.Save = sv
	} else {
		fgt.Save = sv
	}
	if ld == "default" {
		fgt.Load = ld
		fg.Filename = "HPSDRProgrammer_cmd.json"
	} else if ld != "none" {
		fg.Filename = ld
		fgt.Load = ld
	} else {
		fgt.Load = ld
	}

	if fgt.Save != "none" {

		f, err := os.Create(fg.Filename)
		if err != nil {
			panic(err)
		}

		b, err := json.MarshalIndent(fg, "", "\t")
		if err != nil {
			log.Println("error:", err)
		}

		fmt.Fprintf(f, "%s\n", b)
	}

	if ss != "none" {
		Listflags(*fg)
		Listflagstemp(*fgt)
	}

}

func main() {
	var fg flagsettings
	var fgt flagtemp
	//var erstat newopenhpsdr.Erasestatus

	// Create the command line flags
	//ifn := flag.String("interface", "none", "Select one interface number")
	id := flag.Int("index", 0, "Select one interface by number")
	stmac := flag.String("selectMAC", "none", "Select Board by MAC address")
	stip := flag.String("setIP", "none", "Set IP address, unused number from your subnet or 0.0.0.0 for DHCP")
	strbf := flag.String("setRBF", "none", "Select the RBF file to write to the board")
	dd := flag.Int("ddelay", 8, "Discovery delay before a rediscovery")
	ed := flag.Int("edelay", 60, "Discovery delay before a rediscovery")
	db := flag.String("debug", "none", "Turn debugging and output type, (none, dec, hex)")
	ss := flag.String("settings", "none", "Show the settings values (show)")
	sv := flag.String("save", "none", "Save these current flags for future use in default or a named file")
	ld := flag.String("load", "none", "Load a saved command file from default or a named file")
	//cadr := flag.Bool("checkaddress", true, "check if new address is in subdomain and not restricted space")
	//cbad := flag.Bool("checkboard", true, "check if new RBF file name has the same name as the board type")

	flag.Parse()

	if flag.NFlag() < 1 {
		program()
		usage()
	}

	Parseflagstruct(&fg, &fgt, *id, *stmac, *stip, *strbf, *db, *ss, *sv, *ld, *dd, *ed)

	intf := newopenhpsdr.Interfaces()
	for i := range intf {
		if flag.NFlag() < 1 {
			// if no flags list the interfaces in short form
			log.Printf("    %d - %s (%s)\n", intf[i].Index, intf[i].Intname, intf[i].MAC)
		} else if (flag.NFlag() == 1) && (fg.Index == 0) {
			if fg.Debug == "none" {
				// if one flag and it is debug = none, list the interface in short form
				log.Printf("    %d - %s (%s)\n", intf[i].Index, intf[i].Intname, intf[i].MAC)
			} else {
				// if one flag and it is debug = dec or hex, list the interface in long form
				log.Printf("    %d - %s (%s %s  %s\n", intf[i].Index, intf[i].Intname, intf[i].MAC, intf[i].Ipv4, intf[i].Ipv6)
			}
		}

		// if ifn flag matches the current interface
		if fg.Index == intf[i].Index {
			if len(intf[i].Ipv4) != 0 {
				//list the sending computer information
				Listinterface(intf[i])

				var adr string
				var bcadr string
				adr = intf[i].Ipv4 + ":0"
				bcadr = intf[i].Ipv4Bcast + ":1024"

				// perform a discovery
				str, err := newopenhpsdr.Discover(adr, bcadr, fg.Debug)
				if err != nil {
					log.Println("Error ", err)
				}

				//loop throught the list of discovered HPSDR boards
				for i := 0; i < len(str); i++ {
					Listboard(str[i])

					if fg.SelectMAC == str[i].Macaddress {
						log.Printf("      Selected MAC: (%s) %s\n", fg.SelectMAC, str[i].Board)
						crtbd = str[i]

						if (fgt.SetIP != str[i].Baddress) && (fgt.SetIP != "none") {
							//If the IPV4 changes
							if strings.Contains(*stip, "255.255.255.255") {
								log.Printf("     Changing IP address from %s to DHCP address\n\n", str[i].Baddress)
							} else {
								log.Printf("     Changing IP address from %s to %s\n\n", str[i].Baddress, *stip)
							}

							_, err := newopenhpsdr.Setip(adr, bcadr, str[i], *stip, fg.Debug)
							if err != nil {
								log.Printf("Error %v", err)
								panic(err)
							}

							// perform a rediscovery
							time.Sleep(time.Duration(fg.Ddelay) * time.Second)
							str, err = newopenhpsdr.Discover(adr, bcadr, fg.Debug)
							if err != nil {
								log.Println("Error ", err)
							}

							Listboard(str[i])
						} else if *strbf != "none" {
							if (fg.SelectMAC != "none") && (fg.SelectMAC == str[i].Macaddress) {
								if strings.Contains(strings.ToLower(*strbf), strings.ToLower(str[i].Board)) {
									// erase the board flash memory
									//erstat, err := newopenhpsdr.Erase(str[i], fg.SetRBF, fg.Debug)
									//err := newopenhpsdr.Erase(crtbd, fg.Debug)
									err := newopenhpsdr.Erase(adr, str[i], fg.Debug)
									if err != nil {
										panic(err)
									} else {
										//log.Printf(" %v %v\n", erstat.Seconds, erstat.State)
										// send the RBF to the flash memory
										//time.Sleep(8 * time.Second)
										//newopenhpsdr.Program(str[i], fg.SetRBF, fg.Debug)
										err := newopenhpsdr.Program(adr, str[i], *strbf, fg.Debug)
										if err != nil {
											panic(err)
										}
									}
								} else {
									log.Printf("\n      Input Check: RBF name \"%s\" and selectedMAC board name \"%s\" (%s) do not match!\n", *strbf, str[i].Board, str[i].Macaddress)
									log.Printf("       Please correct to program the board.\n")
								}
							} else {
								log.Printf("      Interface not active! \n")
							}
						}
					}
				}
			}
		}
	}
}
