package configs

type Config struct {
    Api   Api    `json:"api"`
    Nodes []Node `json:"nodes"`
}
type Api struct {
    Host string `json:"host"`
    Port int    `json:"port"`
}
type Node struct {
    Name   string `json:"name"`
    Domain int    `json:"domain"`
    Urls   struct {
        Lan []string `json:"lan"`
        Wan []string `json:"wan"`
    } `json:"urls"`
}
