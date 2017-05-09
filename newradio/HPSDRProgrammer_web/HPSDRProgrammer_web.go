// HPSDRProgrammer_web program
//
// by David R. Larsen, copyright December 19, 2015
// License LGPL 2.0
// Part of the OpenHPSDR Software Defined Radio Project (openhpsdr.org)
//
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"os/user"
	"runtime"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/websocket"

	"github.com/TAPR/OpenHPSDR-Protocol2-Programmers/newopenhpsdr"
)

// Server address and port
var srvaddress string

const srvport string = "8228"

// Constants to define the program state
const version string = "0.2.8"
const protocol string = ">1.7"
const update string = "2016-9-17"

//  String to define the program banner in each web window
const banner string = `
<div id="header" align="left" style="background-color:Darkblue;border:2px solid; border-radius:8px; ">
<b align="left" width="90%" style="margin-left:45px; color:white; font-size:38px ">HPSDR Programmer</b>
<p style="margin-left:45px; color:white; font-size:12 ">By Dave, KV&#216S - Version {{.Version}}, Protocol {{.Protocol}} - Last Updated {{.Update}} -  <a style="color:white; font-size:12px" href="http://openhpsdr.org">openhpsdr.org</a> </p>
</div>
`

// HTML to define the style and header settings
const w1p string = `
<html>
<head>
<title> HPSDRProgrammer Web </title>
<meta charset=utf-8 />
`

// HTML to define the style and header settings
const w1style string = `
<style>
html, body, h1, div, select {
  }
  body {
    background: #ffffff;
    color: #000000;
    font-family: Helvetica, Geneva, Arial, sans-serif;
    padding: 20px;
  }
  @-viewport {
    width: 640px;
  }

  h1 {
    font-size: 28px;
    margin-bottom: 20px;
  }
  select, input, button {
		  display:block;
		  border-radius:8px;
		  -moz-border-radius: 8;
		  -webkit-border-radius: 8px;
		  border: 5px solid Darkblue;
		  height: 35px;
		  font-size: 20px;
   }
	table {
	  empty-cells: show
   }
    .hdr {
			display:block;
			background: #eeeeee;
			border-radius:8px;
			-moz-border-radius: 8;
			-webkit-border-radius: 8px;
		   border: 2px solid black;
			height: 35px;
			font-size: 20px;
	 }
	 .nic1 {
		   font-size: 20px;
    }
	 .mac1 {
	   	font-size: 20px;
	 }
	 .btn {
         display:block;
			align:center;
			valign:bottom;
         border-radius:8px;
         -moz-border-radius: 8;
         -webkit-border-radius: 8px;
         border: 5px solid Darkblue;
         height: 35px;
			width: 200px;
         font-size: 20p
    }
	 .intp {
         display:block;
			align:center;
			valign:bottom;
         border-radius:6px;
         -moz-border-radius: 6;
         -webkit-border-radius: 6px;
         border: 3px solid Darkblue;
         height: 35px;
			width: 70;
         font-size: 16
    }
	 .fileintp {
         display:block;
			align:center;
			valign:bottom;
         border-radius:6px;
         -moz-border-radius: 6;
         -webkit-border-radius: 6px;
         border: 3px solid Darkblue;
         height: 35px;
			width: 600;
         font-size: 16
    }
}
</style>
`

// HTML to define the style and header settings
const w2p string = `
</head>
<body>
`

// Intro window text
const intro string = `
<h2>Overview</h2>

<p> The HPSDRProgrammer is a tool to load {{.Protocol}} protocol firmware into HPSDR boards.  This program
performs the same function as the HPSDRProgrammer.  It perform the following tasks.
<ul>
		  <li>Discovery of the HPSDR boards available.</li>
		  <li>Changing the HPSDR board to a fixed IPv4 address within your subnet</li>
		  <li>Erase and Program an RBF file to the HPSDR Board.</li>
</ul>
`

// Counter window text
const w1cnt string = `
<script type="text/javascript" src="http://{{.Address}}:{{.Port}}/js/lib/jquery-1.12.1.min.js"></script>
<script type="text/javascript" >
	var wsUri = "ws://{{.Address}}:{{.Port}}/counter/";
	var output;
	var packet;
	function init() {
			  output = document.getElementById("output");
			  packet = document.getElementById("packet");
			  websocket = new WebSocket(wsUri);
			  websocket.onmessage = function(evt) { onMessage(evt) };
			  websocket.onerror = function(evt) { onError(evt) }; }
   function onMessage(evt) {
			  console.log(evt.data);
			  writeToScreen(evt.data); }
    //function onError(evt) {
	 //		  writeToScreen('<span style="color: red;">ERROR:<\/span> ' + evt.data); }
    function writeToScreen(message) {
			  output.innerHTML = message.split(",")[0];
			  packet.innerHTML = message.split(",")[1]; }
			  window.addEventListener("load", init, false);
</script>
`

const w2cnt string = `
<script>
$(document).ready(function() {
    $.ajax({
		dataType: "json",
		url: "http://{{.Address}}:{{.Port}}/counter/"
    }).then(function(data) {
		data.getJSON(data);
		$('.packet').append(data.packet);
		$('.time').append(data.time);
    });
});
/script>
`

// used get the selected information to different parts of the code with out closure tricks on Handlers
// current selected board
var crtbd newopenhpsdr.Hpsdrboard

// current file
var rbffiledir string
var rbffilename string

//
func usage() {
	log.Printf("    For a list of commands use --help \n\n")
}

// Print out HTML to creat a server computer table
func Computertable(itr newopenhpsdr.Intface) (str string) {
	var strs []string
	s := fmt.Sprintf("<td align=\"right\"><b>Computer:</b></td><td> (%v)</td></tr><tr>", itr.MAC)
	strs = append(strs, s)
	s = fmt.Sprintf("<td align=\"right\"><b>OS:</b></td><td> %s (%s) %d CPU(s)</td></tr><tr>", runtime.GOOS, runtime.GOARCH, runtime.NumCPU())
	strs = append(strs, s)
	if runtime.GOARCH != "arm" {
		u, err := user.Current()
		if err != nil {
			panic(err.Error())
		}
		s = fmt.Sprintf("<td align=\"right\"><b>User:</b></td><td> %s (%s) %s </td></tr><tr>", u.Name, u.Username, u.HomeDir)
		strs = append(strs, s)
	}
	s = fmt.Sprintf("<td align=\"right\"><b>IPV4:</b></td><td> %v</td></tr><tr>", itr.Ipv4)
	strs = append(strs, s)
	s = fmt.Sprintf("<td align=\"right\"><b>IPV6:</b></td><td> %v</td></tr><tr>", itr.Ipv6)
	strs = append(strs, s)
	str = strings.Join(strs, "")
	return str
}

type message struct {
	Time   string `json:"time"`
	Packet string `json:"packet"`
	Msg    chan bool
	Done   chan bool
}

// structure definition for the Html struct
type Html struct {
	Version  string
	Protocol string
	Update   string
	Address  string
	Port     string
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
	log.Printf("              Name: %v\n", itr.Intname)
	log.Printf("               MAC: %v\n", itr.MAC)
	log.Printf("              IPV4: %v\n", itr.Ipv4)
	//log.Printf("              Mask: %d\n", itr.Mask)
	//log.Printf("           Network: %v\n", itr.Network)
	log.Printf("              IPV6: %v\n", itr.Ipv6)
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

// Web handler function to produce the intro webpage
func introhandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Served Introduction page.")

	log.Printf("Browser type %s", r.Header.Get("User-Agent"))

	var H Html
	H.Version = version
	H.Protocol = protocol
	H.Update = update
	H.Address = srvaddress
	H.Port = srvport

	t, _ := template.New("head").Parse(w1p)
	t.Execute(w, "head")

	t1, _ := template.New("style").Parse(w1style)
	t1.Execute(w, "style")

	t2, _ := template.New("body").Parse(w2p)
	t2.Execute(w, "body")

	t3, _ := template.New("webbanner").Parse(banner)
	t3.Execute(w, H)

	t4, _ := template.New("intro").Parse(intro)
	t4.Execute(w, H)

	fmt.Fprintf(w, "<table>")
	fmt.Fprintf(w, "<tr><td>")
	fmt.Fprintf(w, "<form method=\"link\" action=\"/nic/\" >")
	fmt.Fprintf(w, "<button class=\"btn\" type=\"submit\" name=\"nic\" value=\"nic\"> Select Interface</button>")
	fmt.Fprintf(w, "</form>")
	fmt.Fprintf(w, "</td>")
	fmt.Fprintf(w, "<td>")
	fmt.Fprintf(w, "<form method=\"link\" action=\"/closescreen/\" >")
	fmt.Fprintf(w, "<button class=\"btn\" type=\"submit\" name=\"quit\" value=\"quit\"> Quit</button>")
	fmt.Fprintf(w, "</form>")
	fmt.Fprintf(w, "</td></tr>")
	fmt.Fprintf(w, "</table>")

	fmt.Fprintf(w, "</body>\n")
	fmt.Fprintf(w, "</html>\n")

}

// Web handler function to produce the nic selection web page
func nichandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Served Network Interface page.")

	var H Html
	H.Version = version
	H.Protocol = protocol
	H.Update = update
	H.Address = srvaddress
	H.Port = srvport

	r.ParseForm()

	crtbd = newopenhpsdr.ResetHpsdrboard(crtbd)

	str := fmt.Sprintf("http://%s:%s/nic/json/", srvaddress, srvport)

	res, err := http.Get(str)
	if err != nil {
		log.Fatal(err)
	}

	body, err := ioutil.ReadAll(res.Body)

	if err != nil {
		panic(err.Error())
	}

	var intf []newopenhpsdr.Intface
	err = json.Unmarshal(body, &intf)

	t, _ := template.New("head").Parse(w1p)
	t.Execute(w, "head")

	t1, _ := template.New("style").Parse(w1style)
	t1.Execute(w, "style")

	t2, _ := template.New("body").Parse(w2p)
	t2.Execute(w, "body")

	t3, _ := template.New("webbanner").Parse(banner)
	t3.Execute(w, H)

	fmt.Fprintf(w, "<h2>Network Interfaces</h2> <p> Please select the interface to perform a Discovery</p>")

	fmt.Fprintf(w, "<table>\n")
	fmt.Fprintf(w, "<tr><td align=\"right\"><b>Index:</b></td><td><b>(Network) (MAC) (IPV4) (IPV6)</b></td></tr>\n")
	for i := range intf {
		fmt.Fprintf(w, "%s", newopenhpsdr.Intfacetable(intf[i]))
	}
	fmt.Fprintf(w, "</table>\n")
	fmt.Fprintf(w, "<br/>\n")
	fmt.Fprintf(w, "<table><tr><td valign=\"top\">")
	fmt.Fprintf(w, "<form action=\"/board/\" >")
	if r.FormValue("index") == "0" {
		fmt.Fprintf(w, "<b>Select Network interface</b>")
	} else {
		fmt.Fprintf(w, "<b>Selected Network interface</b>")
	}

	fmt.Fprintf(w, "</td><td></td><td></td></tr><tr><td valign=\"top\">")
	fmt.Fprintf(w, "<select name=\"index\" >")
	nic, _ := strconv.ParseInt(r.FormValue("index"), 0, 0)
	for i := range intf {
		if nic == int64(intf[i].Index) {
			fmt.Fprintf(w, "<option selected value=\"%d\">%d: %s (%s)</option>", intf[i].Index, intf[i].Index, intf[i].Intname, intf[i].MAC)
		} else {
			fmt.Fprintf(w, "<option value=\"%d\">%d: %s (%s)</option>", intf[i].Index, intf[i].Index, intf[i].Intname, intf[i].MAC)
		}
	}
	fmt.Fprintf(w, "</select>")
	fmt.Fprintf(w, "</td><td valign=\"top\">")
	fmt.Fprintf(w, "<button class=\"btn\" type=\"submit\" name=\"select\" value=\"nic\"> Select</button>")
	fmt.Fprintf(w, "</form>")

	fmt.Fprintf(w, "</td><td  valign=\"top\">")
	fmt.Fprintf(w, "<form method=\"link\" action=\"/closescreen/\" >")
	fmt.Fprintf(w, "<button class=\"btn\" type=\"submit\" name=\"quit\" value=\"quit\"> Quit</button>")
	fmt.Fprintf(w, "</form>")

	fmt.Fprintf(w, "</td></tr>")
	fmt.Fprintf(w, "</table>")

	fmt.Fprintf(w, "</body>\n")
	fmt.Fprintf(w, "</html>\n")
}

// Web handler function to create the board selection web page
func boardhandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Served Radio Select page.")

	var H Html
	H.Version = version
	H.Protocol = protocol
	H.Update = update
	H.Address = srvaddress
	H.Port = srvport

	t, _ := template.New("head").Parse(w1p)
	t.Execute(w, "head")

	t1, _ := template.New("style").Parse(w1style)
	t1.Execute(w, "style")

	t2, _ := template.New("body").Parse(w2p)
	t2.Execute(w, "body")

	t3, _ := template.New("webbanner").Parse(banner)
	t3.Execute(w, H)

	r.ParseForm()

	fmt.Fprintf(w, "<h2>Computer</h2> ")

	intf := newopenhpsdr.Interfaces()
	var itr newopenhpsdr.Intface

	nic, _ := strconv.ParseInt(r.FormValue("index"), 0, 0)
	for i := range intf {
		if nic == int64(intf[i].Index) {
			itr = intf[i]
		}
	}

	Listinterface(itr)
	fmt.Fprintf(w, "<table>\n")
	fmt.Fprintf(w, "%s", Computertable(itr))
	fmt.Fprintf(w, "</td></tr>\n")
	fmt.Fprintf(w, "</table>\n")

	fmt.Fprintf(w, "<h2>Radios</h2> <p> Please select from these available Radios</p>")
	//var adr string
	//var bcadr string
	var boardtype string
	boardtype = "none"
	//adr = itr.Ipv4 + ":1024"
	//bcadr = itr.Ipv4Bcast + ":1024"

	// perform a discovery
	sd := fmt.Sprintf("http://%s:%s/discover/json/?index=%d", srvaddress, srvport, itr.Index)
	log.Printf("Discovery Call: %s\n", sd)

	res, err := http.Get(sd)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Reply Header: %s\n ", res.Header)
	log.Printf("Reply Body: %s\n ", res.Body)

	body, err := ioutil.ReadAll(res.Body)

	if err != nil {
		log.Println("Read Error ", err)
	}

	var str []newopenhpsdr.Hpsdrboard
	err = json.Unmarshal(body, &str)
	if err != nil {
		log.Println("Unmarshall Error ", err)
	}

	for i := 0; i < len(str); i++ {
		fmt.Fprintf(w, "%s", newopenhpsdr.Hpsdrboardlist(str[i]))
		log.Printf("        %s: (%s) (%s)\n", str[i].Board, str[i].Macaddress, str[i].Baddress)
	}

	fmt.Fprintf(w, "<br/><br/>\n")
	fmt.Fprintf(w, "<table>\n<tr>\n<td align=\"right\">\n")
	fmt.Fprintf(w, "<form action=\"/board/\" >\n")
	if r.FormValue("index") == "0" {
		fmt.Fprintf(w, "<b>Select Network interface</b>")
	} else {
		fmt.Fprintf(w, "<b>Selected Network interface: </b>")
	}

	fmt.Fprintf(w, "</td><td valign=\"top\"> %d: <b>%s</b> (%s)</td><td></td></tr>", itr.Index, itr.Intname, itr.MAC)
	fmt.Fprintf(w, "<tr><td align=\"right\" >")

	if r.FormValue("board") == "" {
		fmt.Fprintf(w, "<b>Select HPSDR Board</b>")
	}
	fmt.Fprintf(w, "</td></tr></table>")

	//fmt.Fprintf(w, "<br/>\n")
	fmt.Fprintf(w, "<table>\n<tr>\n<td align=\"right\" valign=\"top\">\n")
	//fmt.Fprintf(w, "</td><td></td></tr><tr><td>")
	if r.FormValue("index") == "0" {
		fmt.Fprintf(w, "<select disabled=\"disabled\" name=\"board\" >\n")
	} else {
		fmt.Fprintf(w, "<select name=\"board\" >\n")
	}
	fmt.Fprintf(w, "<option value=\"%s\">%s %s</option>\n", "none", "none", "none")
	for i := 0; i < len(str); i++ {
		if r.FormValue("board") == str[i].Macaddress {
			fmt.Fprintf(w, "<option selected value=\"%s\">%s (%s)</option>\n", str[i].Macaddress, str[i].Board, str[i].Macaddress)
			//istr = strings.Split(str[i].Baddress, ".")
			boardtype = str[i].Board
			crtbd = str[i]
		} else {
			fmt.Fprintf(w, "<option value=\"%s\">%s (%s)</option>\n", str[i].Macaddress, str[i].Board, str[i].Macaddress)
		}
	}
	fmt.Fprintf(w, "</select>\n")

	fmt.Fprintf(w, "</td><td valign=\"top\">")

	fmt.Fprintf(w, "<input type=\"hidden\" name=\"index\" value=%s>\n", r.FormValue("index"))
	fmt.Fprintf(w, "<input type=\"hidden\" name=\"boardtype\" value=%s>\n", boardtype)
	fmt.Fprintf(w, "<button class=\"btn\" type=\"submit\" value=\"board\"> Select</button>\n")
	fmt.Fprintf(w, "</form>")

	fmt.Fprintf(w, "</td></tr>")

	fmt.Fprintf(w, "</table>\n")

	if r.FormValue("board") != "" {
		fmt.Fprintf(w, "<b>Select HPSDR Board</b>")
		fmt.Fprintf(w, "<table>\n")
		fmt.Fprintf(w, "%s", newopenhpsdr.Hpsdrboardtable(crtbd))
		Listboard(crtbd)

		fmt.Fprintf(w, "</td><td valign=\"top\">")
		fmt.Fprintf(w, "<form method=\"link\" action=\"/setip/\" >")
		fmt.Fprintf(w, "<input type=\"hidden\" name=\"index\" value=%s>\n", r.FormValue("index"))
		fmt.Fprintf(w, "<input type=\"hidden\" name=\"board\" value=%s>\n", r.FormValue("board"))
		fmt.Fprintf(w, "<input type=\"hidden\" name=\"baddress\" value=%s>\n", crtbd.Baddress)
		fmt.Fprintf(w, "<button class=\"btn\" type=\"submit\" name=\"setip\" value=\"setip\"> Change IP</button>")
		fmt.Fprintf(w, "</form>")
		fmt.Fprintf(w, "</td><td valign=\"top\">")
		fmt.Fprintf(w, "<form method=\"link\" action=\"/prog/\" >")
		fmt.Fprintf(w, "<input type=\"hidden\" name=\"index\" value=%s>\n", r.FormValue("index"))
		fmt.Fprintf(w, "<input type=\"hidden\" name=\"boardtype\" value=%s>\n", boardtype)
		fmt.Fprintf(w, "<input type=\"hidden\" name=\"board\" value=%s>\n", r.FormValue("board"))
		fmt.Fprintf(w, "<button class=\"btn\" type=\"submit\" name=\"program\" value=\"program\"> Program</button>")
		fmt.Fprintf(w, "</form>")
		fmt.Fprintf(w, "</td><td valign=\"top\">")
		fmt.Fprintf(w, "<form method=\"link\" action=\"/closescreen/\" >")
		fmt.Fprintf(w, "<button class=\"btn\" type=\"submit\" name=\"quit\" value=\"quit\"> Quit</button>")
		fmt.Fprintf(w, "</form>")
		fmt.Fprintf(w, "</td></tr>")
		fmt.Fprintf(w, "</table>\n")
		fmt.Fprintf(w, "<br/>\n")
	}

	fmt.Fprintf(w, "</body>\n")
	fmt.Fprintf(w, "</html>\n")

}

// Web handler function to produce the change IP web page.
func setiphandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Served Set IP Interface page.")

	r.ParseForm()

	var H Html
	H.Version = version
	H.Protocol = protocol
	H.Update = update
	H.Address = srvaddress
	H.Port = srvport

	//intf := newopenhpsdr.Interfaces()

	t, _ := template.New("head").Parse(w1p)
	t.Execute(w, "head")

	t1, _ := template.New("style").Parse(w1style)
	t1.Execute(w, "style")

	t2, _ := template.New("body").Parse(w2p)
	t2.Execute(w, "body")

	t3, _ := template.New("webbanner").Parse(banner)
	t3.Execute(w, H)

	adr := r.FormValue("baddress")
	boardtype := r.FormValue("boardtype")
	board := r.FormValue("board")
	nic := r.FormValue("index")

	ad := strings.Split(adr, ":")
	aa := strings.Split(ad[0], ".")
	fmt.Fprintf(w, "<h2>Change IP Interfaces</h2> <p> Please select the interface to perform a Change to the IP address.</p>")
	fmt.Fprintf(w, "<table>\n")
	fmt.Fprintf(w, "</td><td valign=\"top\">")
	fmt.Fprintf(w, "<form method=\"link\" action=\"/changedip/\" >")
	//fmt.Fprintf(w, "<form >")
	s, _ := strconv.ParseInt(aa[0], 10, 32)
	fmt.Fprintf(w, "<input class=\"intp\" type=\"number\" min=\"0\" max=\"254\" name=\"ip1\" value=%d>\n", s)
	fmt.Fprintf(w, "</td><td valign=\"top\">")
	s1, _ := strconv.ParseInt(aa[1], 10, 32)
	fmt.Fprintf(w, "<input class=\"intp\" type=\"number\" min=\"0\" max=\"254\" name=\"ip2\" value=%d>\n", s1)
	fmt.Fprintf(w, "</td><td valign=\"top\">")
	s2, _ := strconv.ParseInt(aa[2], 10, 32)
	fmt.Fprintf(w, "<input class=\"intp\" type=\"number\" min=\"0\" max=\"254\" name=\"ip3\" value=%d>\n", s2)
	fmt.Fprintf(w, "</td><td valign=\"top\">")
	s3, _ := strconv.ParseInt(aa[3], 10, 32)
	fmt.Fprintf(w, "<input class=\"intp\" type=\"number\" min=\"0\" max=\"254\" name=\"ip4\" value=%d>\n", s3)
	fmt.Fprintf(w, "</td><td valign=\"top\">")
	fmt.Fprintf(w, "<input type=\"hidden\" name=\"oldaddress\" value=%s>\n", adr)
	fmt.Fprintf(w, "<input type=\"hidden\" name=\"index\" value=%s>\n", nic)
	fmt.Fprintf(w, "<input type=\"hidden\" name=\"boardtype\" value=%s>\n", boardtype)
	fmt.Fprintf(w, "<input type=\"hidden\" name=\"board\" value=%s>\n", board)
	fmt.Fprintf(w, "<button class=\"btn\" type=\"submit\" name=\"setip\" value=\"setip\"> Set IP</button>")
	fmt.Fprintf(w, "</form>")
	fmt.Fprintf(w, "</td><td valign=\"top\">")
	fmt.Fprintf(w, "<form method=\"link\" action=\"/changedip/\" >")
	fmt.Fprintf(w, "<input type=\"hidden\" name=\"oldaddress\" value=%s>\n", adr)
	fmt.Fprintf(w, "<input type=\"hidden\" name=\"index\" value=%s>\n", nic)
	fmt.Fprintf(w, "<input type=\"hidden\" name=\"boardtype\" value=%s>\n", boardtype)
	fmt.Fprintf(w, "<input type=\"hidden\" name=\"board\" value=%s>\n", board)
	fmt.Fprintf(w, "<button class=\"btn\" type=\"submit\" name=\"dhcp\" value=\"dhcp\"> DHCP</button>")
	fmt.Fprintf(w, "</form>")
	fmt.Fprintf(w, "</td><td valign=\"top\">")
	fmt.Fprintf(w, "<form method=\"link\" action=\"/closescreen/\" >")
	fmt.Fprintf(w, "<button class=\"btn\" type=\"submit\" name=\"quit\" value=\"quit\"> Quit</button>")
	fmt.Fprintf(w, "</form>")

	fmt.Fprintf(w, "</td></tr>")
	fmt.Fprintf(w, "</table>\n")

	fmt.Fprintf(w, "</body>\n")
	fmt.Fprintf(w, "</html>\n")
}

// Web handler function to make the programs board web page.
func prghandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Served Program Interface page.")

	var H Html
	H.Version = version
	H.Protocol = protocol
	H.Update = update
	H.Address = srvaddress
	H.Port = srvport

	r.ParseForm()
	boardtype := r.FormValue("boardtype")
	intf := r.FormValue("index")

	t, _ := template.New("head").Parse(w1p)
	t.Execute(w, "head")
	t1, _ := template.New("style").Parse(w1style)
	t1.Execute(w, "style")

	t2, _ := template.New("body").Parse(w2p)
	t2.Execute(w, "body")

	t3, _ := template.New("webbanner").Parse(banner)
	t3.Execute(w, H)

	fmt.Fprintf(w, "<h2>Program Interfaces</h2> <p> Please select the interface to perform a Board Program</p>")
	fmt.Fprintf(w, "<legend>Get the latest RBF file from the repository</legend>\n")
	if boardtype == "METIS" {
		fmt.Fprintf(w, "<a href=\"http://%s:%s/nosite/\">%s</a><br/>\n", srvaddress, srvport, boardtype)
		//fmt.Fprintf(w, "<a href=\"http://svn.tapr.org/repos_sdr_hpsdr/trunk/Metis/Release/\">%s</a><br/>\n", boardtype)
	} else if boardtype == "HERMES" {
		fmt.Fprintf(w, "<a href=\"http://%s:%s/nosite/\">%s</a><br/>\n", srvaddress, srvport, boardtype)
		//fmt.Fprintf(w, "<a href=\"http://svn.tapr.org/repos_sdr_hpsdr/trunk/Hermes/Release/\">%s</a><br/>\n", boardtype)
	} else if boardtype == "ANGELIA" {
		fmt.Fprintf(w, "<a href=\"http://%s:%s/nosite/\">%s</a><br/>\n", srvaddress, srvport, boardtype)
		//fmt.Fprintf(w, "<a href=\"http://www.k5so.com/HPSDR_downloads.html\">%s</a><br/>\n", boardtype)
	} else if boardtype == "ORIAN" {
		fmt.Fprintf(w, "<a href=\"http://%s:%s/nosite/\">%s</a><br/>\n", srvaddress, srvport, boardtype)
		//fmt.Fprintf(w, "<a href=\"http://www.k5so.com/HPSDR_downloads.html\">%s</a><br/>\n", boardtype)
	}

	fmt.Fprintf(w, "<br/>")

	fmt.Fprintf(w, "<form enctype=\"multipart/form-data\" action=\"/upload/\" method=\"post\">")
	fmt.Fprintf(w, "<input type=\"hidden\" name=\"index\" value=%s>\n", intf)
	fmt.Fprintf(w, "<input type=\"hidden\" name=\"boardtype\" value=%s>\n", boardtype)
	fmt.Fprintf(w, "<input type=\"hidden\" name=\"token\" value=\"{{.}}\">\n")
	fmt.Fprintf(w, " Select a file: <input class=\"fileintp\" type=\"file\" accept=\".rbf, .RBF\" name=\"uploadfile\" id=\"uploadfile\">")
	fmt.Fprintf(w, "  <input class=\"btn\" type=\"submit\" value=\"Upload\">")
	fmt.Fprintf(w, "</form>")
	fmt.Fprintf(w, "<br/>")

	fmt.Fprintf(w, "</body>\n")
	fmt.Fprintf(w, "</html>\n")
}

// Web handler function to make the programs board web page.
func filehandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Served Program Interface page.")

	var H Html
	H.Version = version
	H.Protocol = protocol
	H.Update = update
	H.Address = srvaddress
	H.Port = srvport

	r.ParseForm()

	filename := r.FormValue("img")
	//boardtype := r.FormValue("boardtype")
	//intf := r.FormValue("index")

	t, _ := template.New("head").Parse(w1p)
	t.Execute(w, "head")

	t1, _ := template.New("style").Parse(w1style)
	t1.Execute(w, "style")

	t3, _ := template.New("body").Parse(w2p)
	t3.Execute(w, "body")

	t4, _ := template.New("webbanner").Parse(banner)
	t4.Execute(w, H)

	// Open the RBF file
	rbffilename = filename
	log.Println("    Looking for rbf file:", filename)
	f, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}

	defer func() {
		err := f.Close()
		if err != nil {
			log.Fatal(err)
		}
	}()

	fmt.Fprintf(w, "<table>")
	fmt.Fprintf(w, "<tr>\n")
	fmt.Fprintf(w, "<td align=\"right\"><b class=\"nic1\">Erase flash memory:</b> </td>")
	fmt.Fprintf(w, "<td><div id=\"output\" class=\"nic1\"> </div></td>")
	fmt.Fprintf(w, "</tr><tr>\n")
	fmt.Fprintf(w, "<td align=\"right\"><b class=\"nic1\">Programming:</b> </td>")
	fmt.Fprintf(w, "<td><div id=\"packet\" class=\"nic1\"> </div></td>")
	fmt.Fprintf(w, "</tr>\n")
	fmt.Fprintf(w, "</table><br/><br/>\n")

	t5, _ := template.New("script").Parse(w1cnt)
	t5.Execute(w, H)

	fmt.Fprintf(w, "<table>")
	fmt.Fprintf(w, "<tr><td>")
	fmt.Fprintf(w, "<form method=\"link\" action=\"/nic/\" >")
	fmt.Fprintf(w, "<button class=\"btn\" type=\"submit\" name=\"nic\" value=\"nic\"> Return</button>")
	fmt.Fprintf(w, "</form>")
	fmt.Fprintf(w, "</td>")
	fmt.Fprintf(w, "<td>")
	fmt.Fprintf(w, "<form method=\"link\" action=\"/closescreen/\" >")
	fmt.Fprintf(w, "<button class=\"btn\" type=\"submit\" name=\"quit\" value=\"quit\"> Quit</button>")
	fmt.Fprintf(w, "</form>")
	fmt.Fprintf(w, "</td></tr>")
	fmt.Fprintf(w, "</table>")

	fmt.Fprintf(w, "</body>\n")
	fmt.Fprintf(w, "</html>\n")
}

// Web handler function to produce the quit warning web page.
func nositehandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Served No Site Interface page.")
	var H Html
	H.Version = version
	H.Protocol = protocol
	H.Update = update
	H.Address = srvaddress
	H.Port = srvport

	r.ParseForm()

	t, _ := template.New("head").Parse(w1p)
	t.Execute(w, "head")

	t1, _ := template.New("style").Parse(w1style)
	t1.Execute(w, "style")

	t2, _ := template.New("webbanner").Parse(banner)
	t2.Execute(w, H)

	t3, _ := template.New("body").Parse(w2p)
	t3.Execute(w, "body")

	fmt.Fprintf(w, "<h1>No site at this time!</h1> <p> Please select the Quit or Return</p>")
	fmt.Fprintf(w, "<table>")
	fmt.Fprintf(w, "<tr><td>")
	fmt.Fprintf(w, "<form method=\"link\" action=\"/nic/\" >")
	fmt.Fprintf(w, "<button class=\"btn\" type=\"submit\" name=\"nic\" value=\"nic\"> Return</button>")
	fmt.Fprintf(w, "</form>")
	fmt.Fprintf(w, "</td>")
	fmt.Fprintf(w, "<td>")
	fmt.Fprintf(w, "<form method=\"link\" action=\"/close/\" >")
	fmt.Fprintf(w, "<button class=\"btn\" type=\"submit\" name=\"quit\" value=\"quit\"> Quit</button>")
	fmt.Fprintf(w, "</form>")
	fmt.Fprintf(w, "</td></tr>")
	fmt.Fprintf(w, "</table>")

	fmt.Fprintf(w, "</body>\n")
	fmt.Fprintf(w, "</html>\n")
}

// Web handler function to produce the quit warning web page.
func closescreenhandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Served Close screen Interface page.")
	var H Html
	H.Version = version
	H.Protocol = protocol
	H.Update = update
	H.Address = srvaddress
	H.Port = srvport

	r.ParseForm()

	t, _ := template.New("head").Parse(w1p)
	t.Execute(w, "head")

	t1, _ := template.New("style").Parse(w1style)
	t1.Execute(w, "style")

	t2, _ := template.New("webbanner").Parse(banner)
	t2.Execute(w, H)

	t3, _ := template.New("body").Parse(w2p)
	t3.Execute(w, "body")

	fmt.Fprintf(w, "<h1>Shutting down the HPSDRProgrammer!</h1> <p> Please select the Quit or Return</p>")
	fmt.Fprintf(w, "<table>")
	fmt.Fprintf(w, "<tr><td>")
	fmt.Fprintf(w, "<form method=\"link\" action=\"/nic/\" >")
	fmt.Fprintf(w, "<button class=\"btn\" type=\"submit\" name=\"nic\" value=\"nic\"> Return</button>")
	fmt.Fprintf(w, "</form>")
	fmt.Fprintf(w, "</td>")
	fmt.Fprintf(w, "<td>")
	fmt.Fprintf(w, "<form method=\"link\" action=\"/close/\" >")
	fmt.Fprintf(w, "<button class=\"btn\" type=\"submit\" name=\"quit\" value=\"quit\"> Quit</button>")
	fmt.Fprintf(w, "</form>")
	fmt.Fprintf(w, "</td></tr>")
	fmt.Fprintf(w, "</table>")

	fmt.Fprintf(w, "</body>\n")
	fmt.Fprintf(w, "</html>\n")
}

// Web handler function to stop the webserver.
func closehandler(w http.ResponseWriter, r *http.Request) {
	log.Fatal("Program shut down by user!")
}

func changediphandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Served Changed IP Interface page.")

	var H Html
	H.Version = version
	H.Protocol = protocol
	H.Update = update
	H.Address = srvaddress
	H.Port = srvport

	r.ParseForm()

	nic := r.FormValue("index")
	board := r.FormValue("board")
	oadr := r.FormValue("oldaddress")
	ip1 := r.FormValue("ip1")
	ip2 := r.FormValue("ip2")
	ip3 := r.FormValue("ip3")
	ip4 := r.FormValue("ip4")

	str := fmt.Sprintf("http://%s:%s/setip/json/?index=%s&board=%s&oldaddress=%s&ip1=%s&ip2=%s&ip3=%s&ip4=%s", srvaddress, srvport, nic, board, oadr, ip1, ip2, ip3, ip4)

	res, err := http.Get(str)
	if err != nil {
		log.Fatal(err)
	}

	body, err := ioutil.ReadAll(res.Body)

	if err != nil {
		panic(err.Error())
	}

	var msg newopenhpsdr.SetIPmessage
	err = json.Unmarshal(body, &msg)

	//log.Printf("String %s\n", str)
	//log.Printf("%v\n", msg)

	if r.FormValue("dhcp") == "dhcp" {
		msg.Newaddress = "dhcp"
	}

	t, _ := template.New("head").Parse(w1p)
	t.Execute(w, "head")

	t1, _ := template.New("style").Parse(w1style)
	t1.Execute(w, "style")

	t2, _ := template.New("body").Parse(w2p)
	t2.Execute(w, "body")

	t3, _ := template.New("webbanner").Parse(banner)
	t3.Execute(w, H)

	fmt.Fprintf(w, "<h1>Radio Board changed</h1> ")

	fmt.Fprintf(w, "<table>")
	fmt.Fprintf(w, "<tr><td align=\"right\">")
	fmt.Fprintf(w, "<b>Message:</b>")
	fmt.Fprintf(w, "</td><td> %s", msg.Message)
	fmt.Fprintf(w, "</td></tr><tr><td><b>MAC Address:</b>")
	fmt.Fprintf(w, "</td><td> %s", msg.Macaddress)
	fmt.Fprintf(w, "</td></tr><tr><td><b>Old Address:</b>")
	fmt.Fprintf(w, "</td><td> %s", msg.Oldaddress)
	fmt.Fprintf(w, "</td></tr><tr><td><b>New Address:</b>")
	fmt.Fprintf(w, "</td><td> %s", msg.Newaddress)
	fmt.Fprintf(w, "</td></tr>")
	fmt.Fprintf(w, "</table>")

	fmt.Fprintf(w, "<table>")
	fmt.Fprintf(w, "<tr><td>")
	fmt.Fprintf(w, "<form method=\"link\" action=\"/nic/\" >")
	fmt.Fprintf(w, "<button class=\"btn\" type=\"submit\" name=\"return\" value=\"return\"> Return</button>")
	fmt.Fprintf(w, "</form>")
	fmt.Fprintf(w, "</td>")
	fmt.Fprintf(w, "<td>")
	fmt.Fprintf(w, "<form method=\"link\" action=\"/closescreen/\" >")
	fmt.Fprintf(w, "<button class=\"btn\" type=\"submit\" name=\"quit\" value=\"quit\"> Quit</button>")
	fmt.Fprintf(w, "</form>")
	fmt.Fprintf(w, "</td></tr>")
	fmt.Fprintf(w, "</table>")

	fmt.Fprintf(w, "</body>\n")
	fmt.Fprintf(w, "</html>\n")
}

// Web handler function to produce the nic json packet.
func nicjsonhandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Served Network Interface json.")
	intf := newopenhpsdr.Interfaces()
	enc := json.NewEncoder(w)
	enc.Encode(intf)
}

// Web handler function to produce the discover json packet.
func discoverjsonhandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Served Discovery json.")
	r.ParseForm()

	intf := newopenhpsdr.Interfaces()
	var itr newopenhpsdr.Intface

	nic, _ := strconv.ParseInt(r.FormValue("index"), 0, 0)
	for i := range intf {
		if nic == int64(intf[i].Index) {
			itr = intf[i]
			log.Printf("Match %s  %s %s\n", nic, itr.Intname, itr.Matchname)

		} else {

			log.Printf("No Match %s  (%s) (%s)\n", nic, itr.Intname, itr.Matchname)
		}
	}

	var adr string
	var bcadr string
	adr = itr.Ipv4 + ":1024"
	bcadr = itr.Ipv4Bcast + ":1024"
	log.Printf("adr %s  bcadr %s\n", adr, bcadr)

	str, err := newopenhpsdr.Discover(adr, bcadr, "none")
	if err != nil {
		log.Println("Error ", err)
	}

	enc := json.NewEncoder(w)
	enc.Encode(str)
	log.Printf("%#v\n", intf)
}

// Web handler function to produce the setip json packet.
func setipjsonhandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Served SetIP Interface json.")

	var adr string
	var bcadr string
	var nadr string
	var st newopenhpsdr.Hpsdrboard

	r.ParseForm()

	//fmt.Fprintf(w, "<h2>Computer</h2> ")

	intf := newopenhpsdr.Interfaces()
	var itr newopenhpsdr.Intface

	nic, _ := strconv.ParseInt(r.FormValue("index"), 0, 0)
	for i := range intf {
		if nic == int64(intf[i].Index) {
			itr = intf[i]
		}
	}

	adr = itr.Ipv4 + ":1024"
	bcadr = itr.Ipv4Bcast + ":1024"

	str, err := newopenhpsdr.Discover(adr, bcadr, "none")
	if err != nil {
		log.Println("Error ", err)
	}

	for i := 0; i < len(str); i++ {
		if r.FormValue("board") == str[i].Macaddress {
			st = str[i]
		}
	}
	if r.FormValue("dhcp") == "dhcp" {
		nadr = "0.0.0.0"
		log.Printf("IP changing from %s -> %s", r.FormValue("oldaddress"), r.FormValue("dhcp"))
	} else {
		nadr = fmt.Sprintf("%s.%s.%s.%s", r.FormValue("ip1"), r.FormValue("ip2"), r.FormValue("ip3"), r.FormValue("ip4"))
		log.Printf("IP changing from %s -> %s", r.FormValue("oldaddress"), nadr)
	}

	msg, err := newopenhpsdr.Setip(adr, bcadr, st, nadr, "none")
	if err != nil {
		log.Printf("Error %v", err)
		panic(err)
	}

	enc := json.NewEncoder(w)
	enc.Encode(msg)
}

// Web handler function to produce the setip json packet.
func erasejsonhandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Served Erase Interface json.")
	var msg newopenhpsdr.Erasemessage
	//msg.Message = "Testing Erase"
	//msg.Lapstime = "200"

	enc := json.NewEncoder(w)
	enc.Encode(msg)
}

// Web handler function to produce the setip json packet.
func counthandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Served counter screen Interface page.")
	var H Html
	H.Version = version
	H.Protocol = protocol
	H.Update = update
	H.Address = srvaddress
	H.Port = srvport

	r.ParseForm()

	t, _ := template.New("head").Parse(w1p)
	t.Execute(w, "head")

	t2, _ := template.New("style").Parse(w1style)
	t2.Execute(w, "style")

	t3, _ := template.New("body").Parse(w2p)
	t3.Execute(w, "body")

	t4, _ := template.New("webbanner").Parse(banner)
	t4.Execute(w, H)

	fmt.Fprintf(w, "<table>")
	fmt.Fprintf(w, "<tr>")
	fmt.Fprintf(w, "<td align=\"right\"><b class=\"nic1\">Erase flash memory:</b> </td>")
	fmt.Fprintf(w, "<td><div id=\"output\" class=\"nic1\"> </div></td>")
	fmt.Fprintf(w, "</tr><tr>")
	fmt.Fprintf(w, "<td align=\"right\"><b class=\"nic1\">Programming:</b> </td>")
	fmt.Fprintf(w, "<td><div id=\"packet\" class=\"nic1\"> </div></td>")
	fmt.Fprintf(w, "</tr>")
	fmt.Fprintf(w, "</table>")

	t1, _ := template.New("script").Parse(w1cnt)
	t1.Execute(w, H)

	log.Printf("ws://{{.Address}}:{{.Port}}/counter/")
	//t5, _ := template.New("sct2").Parse(w2cnt)
	//t5.Execute(w, H)

	fmt.Fprintf(w, "</body>")
	fmt.Fprintf(w, "</html>")

}

// Upload the selected file to a common place for use by the programmer
func uploadhandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Served Upload Interface page.")

	var H Html
	H.Version = version
	H.Protocol = protocol
	H.Update = update
	H.Address = srvaddress
	H.Port = srvport

	r.ParseForm()

	//filename := r.FormValue("uploadfile")
	boardtype := r.FormValue("boardtype")
	intf := r.FormValue("index")

	t, _ := template.New("head").Parse(w1p)
	t.Execute(w, "head")

	t1, _ := template.New("style").Parse(w1style)
	t1.Execute(w, "style")

	t3, _ := template.New("body").Parse(w2p)
	t3.Execute(w, "body")

	t4, _ := template.New("webbanner").Parse(banner)
	t4.Execute(w, H)

	r.ParseMultipartForm(32 << 20)
	file, handler, err := r.FormFile("uploadfile")
	if err != nil {
		log.Println(err)
		return
	}
	defer file.Close()

	filestr := fmt.Sprintf("%s%s", rbffiledir, handler.Filename)
	if _, err := os.Stat(rbffiledir); os.IsNotExist(err) {
		os.MkdirAll(rbffiledir, os.ModePerm)
		log.Println("Directory " + rbffiledir + " created")
	} else {
		log.Println("Directory " + rbffiledir + " found")
	}
	log.Println(filestr)

	f, err := os.OpenFile(filestr, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		log.Println(err)
		return
	}
	defer f.Close()
	io.Copy(f, file)

	// Open the RBF file
	rbffilename = filestr
	log.Println("    Looking for rbf file:", filestr)
	f, err = os.Open(filestr)
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
	log.Println("    Found rbf file:", filestr)
	log.Println("     Size rbf file:", fi.Size())
	log.Println("Size rbf in memory:", ((fi.Size()+255)/256)*256)
	log.Println("           Packets:", packets)

	fmt.Fprintf(w, "<h1>Firmware File Information</h1>\n")
	fmt.Fprintf(w, "<table>\n<tr>\n")
	fmt.Fprintf(w, "<td align=\"right\">\n")
	fmt.Fprintf(w, "<b>Found rbf file:</b>")
	fmt.Fprintf(w, "</td><td>")
	fmt.Fprintf(w, "%s", rbffilename)
	fmt.Fprintf(w, "</td></tr>")
	fmt.Fprintf(w, "<td align=\"right\">\n")
	fmt.Fprintf(w, "<b>Size rbf file:</b>")
	fmt.Fprintf(w, "</td><td>")
	fmt.Fprintf(w, "%d", fi.Size())
	fmt.Fprintf(w, "</td></tr>")
	fmt.Fprintf(w, "<td align=\"right\">\n")
	fmt.Fprintf(w, "<b>Size rbf in memory:</b>")
	fmt.Fprintf(w, "</td><td>")
	fmt.Fprintf(w, "%d", (((fi.Size() + 255) / 256) * 256))
	fmt.Fprintf(w, "</td></tr>")
	fmt.Fprintf(w, "<td align=\"right\">\n")
	fmt.Fprintf(w, "<b>Packets:</b>")
	fmt.Fprintf(w, "</td><td>")
	fmt.Fprintf(w, "%d", packets)
	fmt.Fprintf(w, "</td></tr>")
	fmt.Fprintf(w, "</table><br/><br/>\n")
	fmt.Fprintf(w, "<form action=\"/file/\">")
	fmt.Fprintf(w, "<input type=\"hidden\" name=\"index\" value=%s>\n", intf)
	fmt.Fprintf(w, "<input type=\"hidden\" name=\"boardtype\" value=%s>\n", boardtype)
	fmt.Fprintf(w, "<input type=\"hidden\" name=\"img\" value=%s>\n", filestr)
	fmt.Fprintf(w, "<input class=\"btn\" type=\"submit\" value=\"Program\">")
	fmt.Fprintf(w, "</form>")
	fmt.Fprintf(w, "<br/>")

	fmt.Fprintf(w, "</body>\n")
	fmt.Fprintf(w, "</html>\n")

}

func sensorhandler(ws *websocket.Conn) {
	var m chan int
	var mnum int
	var msge string
	var seccount int
	var erasing bool
	m = make(chan int)
	seccount = -1

	go func() {
		readsensor(m, rbffiledir, rbffilename)
	}()

	for {
		select {
		case mnum = <-m:
			if mnum == 2001 {
				erasing = true
				msge = fmt.Sprintf("Erase Started 0 seconds, Pending")
			} else if mnum == 2999 {
				erasing = false
				seccount = 0
				msge = fmt.Sprintf("Erase Done, Pending")
			} else if mnum == 1003 {
				msge = fmt.Sprintf("Erase Done, Programming Started 0 Seconds")
			} else if mnum == 1004 {
				msge = fmt.Sprintf("Erase Done, Programmming Done")
			}
		default:
			if erasing {
				seccount += 1
				msge = fmt.Sprintf("Erase %4.1f seconds, Pending", float64(seccount)/100.0)
			} else {
				seccount += 1
				msge = fmt.Sprintf("Erase Done, Programming Started %4.1f seconds", float64(seccount)/100.0)
			}
		}
		websocket.Message.Send(ws, msge)
		time.Sleep(10 * time.Millisecond)
		if mnum == 1004 {
			break
		}
	}
}

func readsensor(m chan int, rdir string, rfile string) {
	m <- 2001

	err := newopenhpsdr.Erase(crtbd.Pcaddress, crtbd, "none")
	if err != nil {
		log.Fatal(err)
	} else {
		m <- 2999

		//fullfilename := rdir + rfile
		log.Printf("Reading RBF file %s\n", rbffilename)
		m <- 1003
		err = newopenhpsdr.Program(crtbd.Pcaddress, crtbd, rbffilename, "none")
		if err != nil {
			log.Fatal(err)
		}
		m <- 1004
	}
}

// Main function for the HPSDRProgrammer_web program.
func main() {
	strbfdir := flag.String("setRBFdir", "none", "Select the RBF Directory")
	address := flag.String("address", "localhost", "Select server IP address")

	flag.Parse()

	if flag.NFlag() < 1 {
		usage()
	}

	if *strbfdir == "none" {
		if runtime.GOARCH != "arm" {
			u, err := user.Current()
			if err != nil {
				log.Println(err)
			}
			if runtime.GOOS == "windows" {
				rbffiledir = fmt.Sprintf("%s\\Downloads\\HPSDRfiles\\", u.HomeDir)
			} else {
				rbffiledir = fmt.Sprintf("%s/Downloads/HPSDRfiles/", u.HomeDir)
			}
		} else {
			rbffiledir = fmt.Sprintf("/home/pi/HPSDRfiles/")
		}
	} else {
		rbffiledir = *strbfdir
	}

	log.Printf("RBF directory %s", rbffiledir)

	log.Println("Listening ...")

	if *address == "localhost" {
		log.Println("Point your web browser to: http://localhost:8228/intro/ ")
		srvaddress = *address
	} else {
		log.Printf("Point your web browser to: http://%s:8228/intro/ ", *address)
		srvaddress = *address
	}

	http.HandleFunc("/nic/json/", nicjsonhandler)
	http.HandleFunc("/setip/json/", setipjsonhandler)
	http.HandleFunc("/discover/json/", discoverjsonhandler)
	http.HandleFunc("/erase/json/", erasejsonhandler)
	//http.HandleFunc("/packet/json/", packetjsonhandler)
	//http.HandleFunc("/program/json/", programjsonhandler)
	http.HandleFunc("/prog/", prghandler)
	http.HandleFunc("/file/", filehandler)
	http.HandleFunc("/upload/", uploadhandler)
	http.HandleFunc("/nic/", nichandler)
	http.HandleFunc("/board/", boardhandler)
	http.HandleFunc("/setip/", setiphandler)
	http.HandleFunc("/changedip/", changediphandler)
	http.HandleFunc("/closescreen/", closescreenhandler)
	http.HandleFunc("/close/", closehandler)
	http.HandleFunc("/nosite/", nositehandler)
	http.Handle("/counter/", websocket.Handler(sensorhandler))
	http.HandleFunc("/count/", counthandler)
	http.HandleFunc("/intro/", introhandler)
	http.Handle("/js/", http.FileServer(http.Dir(".")))

	lsnadr := fmt.Sprintf(":%s", srvport)

	log.Fatal(http.ListenAndServe(lsnadr, nil))
}
