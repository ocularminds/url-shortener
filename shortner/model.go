package shortner

import "time"

//Config application configuration
type Config struct {
    Port   int `json:"port"`
    Db     Db  `json:"database"`
}

//Db database connection properties
type Db struct {
    Name       string `json:"name"`
    UserName   string `json:"username"`
    Password   string `json:"password"`
    Host       string `json:"host"`
    Port       int    `json:"port"`
}

//ShortLink models the shortend url information
type ShortLink struct {
	Shortened string
	Original  string `json:"url"`
	Expiry    int
	Created   time.Time
	Hits      int
}
