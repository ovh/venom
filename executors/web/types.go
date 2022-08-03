package web

// Action represents what can be done with web executor
type Action struct {
	Click           *Click       `yaml:"click,omitempty"`
	Fill            []Fill       `yaml:"fill,omitempty"`
	Find            interface{}  `yaml:"find,omitempty"`
	Navigate        *Navigate    `yaml:"navigate,omitempty"`
	Wait            int64        `yaml:"wait,omitempty"`
	ConfirmPopup    bool         `yaml:"confirmPopup,omitempty"`
	CancelPopup     bool         `yaml:"cancelPopup,omitempty"`
	Select          *Select      `yaml:"select,omitempty"`
	UploadFile      *UploadFile  `yaml:"uploadFile,omitempty"`
	SelectFrame     *SelectFrame `yaml:"selectFrame,omitempty"`
	SelectRootFrame bool         `yaml:"selectRootFrame,omitempty"`
	NextWindow      bool         `yaml:"nextWindow,omitempty"`
	HistoryAction   string       `yaml:"historyAction,omitempy"`
	Execute         *Execute     `yaml:"execute,omitempy"`
}

// Fill represents informations needed to fill input/textarea
type Fill struct {
	Find        interface{} `yaml:"find,omitempty"`
	Text        string      `yaml:"text,omitempty"`
	Key         *string     `yaml:"key,omitempty"`
	SyncTimeout int64       `yaml:"syncTimeout"`
}

// Click represents informations needed to click on web components
type Click struct {
	Find        interface{} `yaml:"find,omitempty"`
	Wait        int64       `yaml:"wait"`
	SyncTimeout int64       `yaml:"syncTimeout"`
}

// Navigate represents informations needed to navigate on defined url
type Navigate struct {
	URL   string `yaml:"url,omitempty"`
	Reset bool   `yaml:"reset,omitempty"`
}

// Select represents informations needed to select an option
type Select struct {
	Find        interface{} `yaml:"find,omitempty"`
	Text        string      `yaml:"text,omitempty"`
	Wait        int64       `yaml:"wait,omitempty"`
	SyncTimeout int64       `yaml:"syncTimeout"`
}

// UploadFile represents informations needed to upload files
type UploadFile struct {
	Find        interface{} `yaml:"find,omitempty"`
	Files       []string    `yaml:"files,omitempty"`
	Wait        int64       `yaml:"wait,omitempty"`
	SyncTimeout int64       `yaml:"syncTimeout"`
}

// SelectFrame represents informations needed to select the frame
type SelectFrame struct {
	Find        interface{} `yaml:"find,omitempty"`
	SyncTimeout int64       `yaml:"syncTimeout"`
}

// Execute represents informations required to execute Javascript
type Execute struct {
	Command string   `yaml:"command,omitempty"`
	Args    []string `yaml:"args,omitempty"`
}
