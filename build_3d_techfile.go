// Generate a techfile for GDS3D from a PDK files 

// Author: Jørgen Kragh Jakobsen
// Date  : 10 Aug 2024

// Pass klayout lyp file go get a list of layername with gds layer number, datatype and color
// Pass lef file to get a layer name with height (z-level) and layer thickness 
// Some layer do not have a height and thickness specified but must be canculated from the stackup

// The techfile is a text file with the following format
/* 

LayerStart: Substrate
Layer: 255
Datatype: 0
Height: -10000.0
Thickness: 10000.0
Red: 0.15
Green: 0.15
Blue: 0.15
Filter: 0.0
Metal: 0
Show: 1
LayerEnd
*/ 

// Lef file from pdk in   IHP-Open-PDK/ihp-sg13g2/libs.ref/sg13g2_stdcell/lef/sg13g2_tech.lef
// Klayout config in      IHP-Open-PDK/ihp-sg13g2/libs.tech/klayout/tech/sg13g2.lyp


package main 

// 

import (
	"fmt"
	"os"
	"time"
	"bufio"
	"strconv"
	"strings" 
	"encoding/xml"
)

// Layer represents a layer with its name, number, and color

type KLayer struct {
	Name    string `xml:"name"`
	Number  string `xml:"source"`
	Color   string `xml:"fill-color"`
	XMLName xml.Name `xml:"properties"`
}
// LayerProperties represents the root element of the XML file

type KLayerProperties struct {
	XMLName   xml.Name `xml:"layer-properties"`
	Properties []KLayer `xml:"properties"`
}

func parseLypFile(filePath string) ([]KLayer, error) {
	// Open the XML file
	file, err := os.Open(filePath)
	if err != nil {
			return nil, err
	}
	defer file.Close()

	// Decode the XML file into a LayerProperties struct
	decoder := xml.NewDecoder(file)
	var layerProps KLayerProperties
	err = decoder.Decode(&layerProps)
	if err != nil {
			return nil, err
	}

	// Filter layers with type "drawing"
	var layers []KLayer
	for _, prop := range layerProps.Properties {
			if _, ok := splitLayerName(prop.Name); ok {
					layers = append(layers, prop)
			}
	}

	return layers, nil
}

func splitLayerName(name string) (string, bool) {
	parts := strings.Split(name, ".")
	if len(parts) != 2 || parts[1] != "drawing" {
			return "", false
	}
	return parts[0], true
}
	
type LefLayer struct {
    Name        string
    Type        string
    Thickness   float64
    Height      float64
}

type LEFFile struct {
    Layers []LefLayer
	Version float64
	DividerChar string
}

func tokenize(line string) []string {
    return strings.Fields(line)
}


const (
	MODE_IDLE = iota
	MODE_UNITS
	MODE_LAYER
	MODE_LAYER_IGNORE
	MODE_VIA
	MODE_VIA_IGNORE
)

func contains(s []string, str string) bool {
    for _, v := range s {
        if v == str {
            return true
        }
    }
    return false
} 

func parseLEF(filePath string) (*LEFFile, error) {

	deflayers := []string{"GatPoly", "Cont", "Metal1", "Via1", "Metal2", "Via2", "Metal3", "Via3", "Metal4", "Via4", "Metal5", "TopVia1", "TopMetal1", "TopVia2", "TopMetal2"}

	mode  := MODE_IDLE

	file, err := os.Open(filePath)
    if err != nil {
        return nil, err
    }
    defer file.Close()

    scanner := bufio.NewScanner(file)
    lefFile := &LEFFile{}

    currentLayer := LefLayer{}
    
    for scanner.Scan() {
        line := scanner.Text()
        tokens := tokenize(line)
        if len(tokens) == 0 {
            continue
        }
		
		// Find section and simple key value pairs
		switch mode { 
		case MODE_IDLE : 
			switch tokens[0] {
		
			case "VERSION": 
				version, err := strconv.ParseFloat(tokens[1], 64) 
				if err == nil {
					lefFile.Version = version
				    fmt.Println("Found version: ", lefFile.Version)
				}
				mode = MODE_IDLE 
			case "DIVIDERCHAR": 
			    lefFile.DividerChar = tokens[1]
				mode = MODE_IDLE 
			case "UNITS":
				mode = MODE_UNITS
				fmt.Println("Found units: ", mode)
			case "LAYER":
				if contains(deflayers,tokens[1]) {
					fmt.Println("Found layer: ", tokens[1])
					currentLayer = LefLayer{Name: tokens[1]}			
					mode = MODE_LAYER
				} else {
					//fmt.Printf("Layer not in default layers: %s (Ignore)\n", tokens[1])	
					mode = MODE_LAYER_IGNORE
				}	
 			
			case "Via":
				mode = MODE_VIA_IGNORE
				//fmt.Printf("Found via: %s (ignore)\n", tokens[1])
			    
			case "ViaRULE":
				mode = MODE_VIA_IGNORE
				//fmt.Printf("Found viaRULE: %s (ignore)\n", tokens[1])
			    
			}
		case MODE_UNITS:
			switch tokens[0] { 
			case "END": 
			 	mode = MODE_IDLE
			    fmt.Println("End of units: ", mode)
			}
		case MODE_LAYER:
			switch tokens[0] {
			case "TYPE":
                currentLayer.Type = tokens[1]
            case "THICKNESS":
                thickness, err := strconv.ParseFloat(tokens[1], 64)
                if err == nil {
                    currentLayer.Thickness = thickness
                }
            case "HEIGHT":
                height, err := strconv.ParseFloat(tokens[1], 64)
                if err == nil {
                    currentLayer.Height = height
                }
            case "END":
                lefFile.Layers = append(lefFile.Layers, currentLayer)
                mode = MODE_IDLE
            }
        case MODE_LAYER_IGNORE:
		    switch tokens[0] {
		        case "END":
		   	    mode = MODE_IDLE
	   	    }
	    
	    case MODE_VIA_IGNORE:
		    switch tokens[0] {
		        case "END":
		   	    mode = MODE_IDLE
	   	    }
	    }
	}	

    if err := scanner.Err(); err != nil {
        return nil, err
    }

    return lefFile, nil
   

}
 
type Layer struct { 
	Name string
	altName string
	GDSNumber int
	GDSDatatype int
	Color string
	Height float64
	Thickness float64
	Metal int
}




func main() {
	
	LayerStack := []Layer{ 	{ "Substrate", 	"Substrate", 255, 0, "#FFFFFF", -10.0, 10.0, 0},
							{ "NWell", 		"NWell",     0, 0, "#000000", 0.0, 0.2,    0},
							{ "PWell", 		"PWell",     0, 0, "#000000", 0.0, 0.2,    0},
							{ "Active", 	"Active",    0, 0, "#000000", 0.2, 0.12,   0},
							{ "ResPoly", 	"ResPoly",   0, 0, "#000000", 0.32, 0.1,   0},
							{ "GatPoly", 	"GatPoly",   0, 0, "#FF0000", 0.32, 0.1,   0},
							{ "Cont", 		"Cont",      0, 0, "#00FF00", 0.32, 0.64,  0},
							{ "Metal1", 	"Metal1",    0, 0, "#0000FF", 0.0, 0.0,    1},
							{ "Via1", 		"Via1",      0, 0, "#FFFF00", 0.0, 0.0,    0},
							{ "Metal2", 	"Metal2",    0, 0, "#00FFFF", 0.0, 0.0,    1},
							{ "Via2", 		"Via2",      0, 0, "#FF00FF", 0.0, 0.0,    0},
							{ "Metal3", 	"Metal3",    0, 0, "#FF0000", 0.0, 0.0,    1},
							{ "Via3", 		"Via3",      0, 0, "#00FF00", 0.0, 0.0,    0},
							{ "Metal4", 	"Metal4",    0, 0, "#0000FF", 0.0, 0.0,    1},
							{ "Via4", 		"Via4",      0, 0, "#FFFF00", 0.0, 0.0,    0},
							{ "Metal5", 	"Metal5",   0, 0, "#00FFFF", 0.0, 0.0,    1},
							{ "TopVia1", 	"TopVia1",  0, 0, "#FF00FF", 0.0, 0.0,    0},
							{ "TopMetal1",  "TopMetal1",0, 0, "#FF0000", 0.0, 2.0,    1},
							{ "TopVia2", 	"TopVia2",  0, 0, "#00FF00", 0.0, 0.0,    0},
							{ "TopMetal2",  "TopMetal2",0, 0, "#0000FF", 0.0, 3.0,    1},
							{ "MIM", 		"MIM",	    0, 0, "#00FFFF", 5.3, 0.150,  0},
    }						
  							
	filePath := "sg13g2.lyp" // Replace with your file path
	layers, err := parseLypFile(filePath)
	if err != nil {
		fmt.Println("Error parsing Lyp file:", err)
		return
	}

	for _, layer := range layers {
		fmt.Printf("Layer name: %s, Number: %s, Color: %s\n", layer.Name, layer.Number, layer.Color)
		update_layerstack(LayerStack,layer)	 
	}

	lefFile, err := parseLEF("sg13g2_tech.lef")
    if err != nil {
        fmt.Println("Error parsing LEF file:", err)
        return
    }

    for _, layer := range lefFile.Layers {
        fmt.Printf("Layer: %s, Type: %s, Thickness: %f, Height: %f\n", layer.Name, layer.Type, layer.Thickness, layer.Height)
		if layer.Thickness > 0.0 {
			update_layerstack_height(LayerStack,layer)
		}
	}

    update_layerstack_vias( LayerStack )
	writeTechFile(LayerStack )
}

func update_layerstack_vias(LayerStack []Layer) {
	for i, l := range LayerStack {
		if (strings.Contains(l.Name, "Via")) && (LayerStack[i].Thickness == 0.0) { 
			LayerStack[i].Height = LayerStack[i-1].Height + LayerStack[i-1].Thickness
			LayerStack[i].Thickness = LayerStack[i+1].Height - LayerStack[i].Height
		    fmt.Printf("Layer: %s, Height: %f, Thickness: %f\n", LayerStack[i].Name, LayerStack[i].Height, LayerStack[i].Thickness) 
		}
	}
}


func update_layerstack(LayerStack []Layer, layer KLayer) {
	for i, l := range LayerStack {
		name := strings.Split(layer.Name, ".")[0]
		if name == l.Name {
			// Split gdsnumber into gds and layertype	
			gdslayertype := strings.Split(layer.Number, "/")
			LayerStack[i].GDSNumber   , _  = strconv.Atoi(gdslayertype[0])
			LayerStack[i].GDSDatatype , _  = strconv.Atoi(gdslayertype[1])
			
			// Copy color string 
			LayerStack[i].Color = layer.Color
			fmt.Printf("Layer: %s, Number: %s, Color: %s\n", LayerStack[i].Name, layer.Number, LayerStack[i].Color)
			fmt.Printf("Layer: %s, Number: %s, Color: %s\n", LayerStack[i].Name, layer.Number, layer.Color)
		}
	}
}

func update_layerstack_height(LayerStack []Layer, layer LefLayer) {
	for i, l := range LayerStack {
		if l.Name == layer.Name {
			LayerStack[i].Height = layer.Height
			LayerStack[i].Thickness = layer.Thickness
		}
	}
}


func writeTechFile(LayerStack []Layer) {
	file, err := os.Create("sg13g2.txt")
	if err != nil {
		fmt.Println("Error creating file:", err)
		return
	}	
	defer file.Close()

	writeTechFileHeader(file)

	for _, layer := range LayerStack {
		writeLayer(file, layer)
	}
}



func writeTechFileHeader(file *os.File) {
	file.WriteString("# Autogenerated GDS3D techfile \n") 
	file.WriteString("# Process : IHP 130nm open source \n")
	file.WriteString("# Author  : Jørgen Kragh Jakobsen \n")
	now := time.Now()
    formattedTime := now.Format("2006-01-02 15:04:05")
	file.WriteString("# Date    : " + formattedTime + "\n")
	file.WriteString("# \n")
	file.WriteString("# Copyright (C) 2024 Jorgen Kragh Jakobsen <jkj@icworks.dk>\n")
	file.WriteString("# \n")
	file.WriteString("# This program is free software; you can redistribute it and/or modify it\n")
	file.WriteString("# under the terms of the GNU General Public License as published by the Free\n")
	file.WriteString("# Software Foundation; either version 2 of the License, or (at your option)\n")
	file.WriteString("# any later version.\n")
	file.WriteString("# \n")
	file.WriteString("# This program is distributed in the hope that it will be useful, but WITHOUT\n")
	file.WriteString("# ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or\n")
	file.WriteString("# FITNESS FOR A PARTICULAR PURPOSE. See the GNU General Public License for\n")
	file.WriteString("# more details.\n")
	file.WriteString("# \n")
	file.WriteString("# You should have received a copy of the GNU General Public License along with\n")
	file.WriteString("# this program; if not, write to the Free Software Foundation, Inc., 51\n")
	file.WriteString("# Franklin Street, Fifth Floor, Boston, MA 02110-1301, USA.\n")
	file.WriteString("# \n")
	file.WriteString("# SPDX-License-Identifier: GPL-2.0-or-later\n\n")
} 			


func writeLayer(file *os.File, layer Layer) {
   	file.WriteString("LayerStart: " + layer.Name + "\n")
	GDSNumber := strconv.Itoa(layer.GDSNumber) 
	if layer.Name == "Substrate" {	
		GDSNumber = "255" 
	} 
	file.WriteString("Layer: " + GDSNumber + "\n")
	file.WriteString("Datatype: " + strconv.Itoa(layer.GDSDatatype) + "\n")
	height_str := fmt.Sprintf("%.0f",layer.Height*1000.0)
	file.WriteString("Height: " +  height_str + "\n")
	thickness_str := fmt.Sprintf("%.0f",layer.Thickness*1000.0)	
	file.WriteString("Thickness: " + thickness_str + "\n")
	red_int , _ := strconv.ParseInt(layer.Color[1:3], 16, 64)
	fmt.Printf("Red: %s -> %d \n", layer.Color[1:3], red_int)	
	red_float 	:= (float64(red_int) / 255.0)
	red_str 	:= fmt.Sprintf("%0.2f",red_float) 
	fmt.Printf("Red: %s \n", red_str)	
	
	green_int , _ := strconv.ParseInt(layer.Color[3:5], 16, 64)
	green_float   :=  (float64(green_int) / 255.0 )
	green_str 	  := fmt.Sprintf("%0.2f",green_float) 
	
	blue_int ,  _  := strconv.ParseInt(layer.Color[5:7], 16, 64)
	blue_float    := (float64(blue_int) / 255.0 ) 
	blue_str 	  := fmt.Sprintf("%0.2f",blue_float) 

	file.WriteString("Red: " + red_str + "\n")
	file.WriteString("Greeen: " + green_str + "\n")
	file.WriteString("Blue: " + blue_str + "\n")
	file.WriteString("Filter: 0.0\n")
	file.WriteString("Metal: " + strconv.Itoa(layer.Metal) + "\n")
	file.WriteString("Show: 1\n")
	file.WriteString("LayerEnd\n\n")
}

