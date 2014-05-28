package main

import(
  "net/http"
  "encoding/json"
  "io/ioutil"
  "os/exec"
  "os"
  "log"
  "errors"
  "strconv"
)

//GitlabRepository represents repository information from the webhook
type GitlabRepository struct {
  Name, Url, Description, Home string
}

//Commit represents commit information from the webhook
type Commit struct {
  Id, Message, Timestamp, Url string
  Author Author
}

//Author represents author information from the webhook
type Author struct {
  Name, Email string
}

//Webhook represents push information from the webhook
type Webhook struct {
  Before, After, Ref, User_name string
  User_id, Project_id int
  Repository GitlabRepository
  Commits []Commit
  Total_commits_count int
}

//ConfigRepository represents a repository from the config file
type ConfigRepository struct {
  Name string
  Commands []string
}

//Config represents the config file
type Config struct {
  Logfile string
  Address string
  Port int64
  Repositories []ConfigRepository
}

func PanicIf(err error, what ...string) {
  if(err != nil) {
    if(len(what) == 0) {
      panic(err)
    }
    
    panic(errors.New(err.Error() + what[0]))
  }
}

var config Config

func main() {
  //load config
  config := loadConfig()

  //open log file
  writer, err := os.OpenFile(config.Logfile, os.O_RDWR|os.O_APPEND, 0666)
  PanicIf(err)
  
  //close logfile on exit
  defer func() {
    writer.Close()
  }()

  //setting logging output
  log.SetOutput(writer)

  //setting handler
  http.HandleFunc("/", hookHandler)

  address := config.Address + ":" + strconv.FormatInt(config.Port, 10)

  log.Println("Listening on " + address)
  
  //starting server
  err = http.ListenAndServe(address, nil)
  if(err != nil) {
    log.Println(err)
  }
}

func loadConfig() Config {
  var file, err = os.Open("config.json")
  PanicIf(err)

  // close file on exit and check for its returned error
  defer func() {
      err := file.Close()
      PanicIf(err)
  }()

  buffer := make([]byte, 1024)
  count := 0

  count, err = file.Read(buffer)
  PanicIf(err)

  err = json.Unmarshal(buffer[:count], &config)
  PanicIf(err)

  return config
}

func hookHandler(w http.ResponseWriter, r *http.Request) {
  defer func() {
    if r := recover(); r != nil {
      log.Println(r)
    }
  }()
  
  var hook Webhook

  //read request body
  var data, err = ioutil.ReadAll(r.Body)
  PanicIf(err, "while reading request")

  //unmarshal request body
  err = json.Unmarshal(data, &hook)
  PanicIf(err, "while unmarshaling request")

  //find matching config for repository name
  for _, repo := range config.Repositories {
    if(repo.Name != hook.Repository.Name) { continue }
    
    //execute commands for repository
    for _, cmd := range repo.Commands {
      var command = exec.Command(cmd)
      err = command.Run()
      if(err != nil) {
        log.Println(err)
      } else {
        log.Println("Executed: " + cmd)
      }
    }
  }
}