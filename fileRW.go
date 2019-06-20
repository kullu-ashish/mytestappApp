package fileRW

import (
    "encoding/json"
    "fmt"
    "io/ioutil"
	"os"
	"net/http"
	
)


// init path
type Init struct {
	Path string
}

// Users struct which contains
// an array of users
/*
type Machines struct {
    Machines []Machine `json:"machines"`
}
*/
var Machines []*Machine

// User struct which contains a name
// a type and a list of social links
type Machine struct {
    Name      string `json:"name"`
    Folders	  []Folder `json:"folders"`
}

type Folder struct {
    Path      string `json:"path"`
    Cron	  string `json:"cronExp"`
}

func init() {
	fmt.Println("init 1")
	Machines = make([]*Machine, 5)
}

func (m *Machine) Render(w http.ResponseWriter, r *http.Request) error {
	// Pre-processing before a response is marshalled and sent across the wire
	return nil
}

func (f *Init) WriteFile(data []*Machine) {

	file, _ := json.MarshalIndent(data, "", " ")
 	_ = ioutil.WriteFile(f.Path, file, 0644)
}

func (f *Init) ReadFile() ([]*Machine, error) {
// Open our jsonFile
jsonFile, err := os.Open(f.Path)
// if we os.Open returns an error then handle it
if err != nil {
	fmt.Println(err)
	return nil, err
}

// defer the closing of our jsonFile so that we can parse it later on
defer jsonFile.Close()

// read our opened xmlFile as a byte array.
byteValue, _ := ioutil.ReadAll(jsonFile)

// we initialize our Users array
var machines []*Machine

// we unmarshal our byteArray which contains our
// jsonFile's content into 'users' which we defined above
json.Unmarshal(byteValue, &machines)

return machines, nil

}

func main() {

	F1 := Folder {"C:\\Temp", "0 0 0 1 1 ? 1970"}
	F2 := Folder {"C:\\Delta", "0 0 0 1 1 ? 1970"}

	F3 := Folder {"C:\\Delta1", "0 0 0 1 1 ? 1970"}
	F4 := Folder {"C:\\Delta2", "0 0 0 1 1 ? 1970"}
	


	M1 := Machine {
		Name: "ZTSQL01",
		Folders: []Folder{
			F1, 
			F2,
		},
	}

	M2 := Machine {
		Name: "ZTSQL02",
		Folders: []Folder{
			F3,
			F4,
		},
	}

	f := Init {"folders.json"}
	mArray := []*Machine{&M1, &M2,}

	f.WriteFile(mArray)

	
	mArray, _ = f.ReadFile()
	fmt.Println("Successfully Opened folders.json")
	
	
    // we iterate through every user within our users array and
    // print out the user Type, their name, and their facebook url
    // as just an example
    for i := 0; i < len(mArray); i++ {
		fmt.Println("Machine: " + mArray[i].Name)
		for j := 0; j < len(mArray[i].Folders); j++ {
			fmt.Println("folder: " + mArray[i].Folders[j].Path)
			fmt.Println("folder: " + mArray[i].Folders[j].Cron)
		}
    }

}