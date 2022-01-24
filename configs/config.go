package configs

type Config struct {
    Api   Api    `json:"api"`
    Nodes []Node `json:"nodes"`
}
type Api struct {
    Host           string `json:"host"`
    ShortPort      string `json:"short_port"`
    PersistentPort string `json:"persistent_port"`
}
type Node struct {
    Name    string  `json:"name"`
    Domain  int     `json:"domain"`
    Retry   int     `json:"retry"`
    Timeout int     `json:"timeout"`
    Account Account `json:"account"`
    Urls    struct {
        Lan []string `json:"lan"`
        Wan []string `json:"wan"`
    } `json:"urls"`
}

type Account struct {
    Account    string `json:"account"`
    PublicKeys string `json:"public-keys"`
}
