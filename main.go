package main

import (
    "encoding/json"
    "fmt"
    "io/ioutil"
)
import "ocularminds.com/shortner"

func main() {

   data, err := ioutil.ReadFile("config.json")
    if err != nil {
        fmt.Println(err)
    }   
    var config shortner.Config
    err = json.Unmarshal(data, &config)
    if err != nil {
        fmt.Println("error converting json ",err)
    }   
    fmt.Println("Server Port:",config.Port,"database:", config.Db.Name) 
    shortner.BuildRoutes(config)
}
