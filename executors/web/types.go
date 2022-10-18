package web

// Action represents what can be done with web executor
type Action struct {
	Click           *Click       `yaml:"click,omitempty"`
	Fill            []Fill       `yaml:"fill,omitempty"`
	Find            string       `yaml:"find,omitempty"`
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
}

// Fill represents information needed to fill input/textarea
type Fill struct {
	Find string  `yaml:"find,omitempty"`
	Text string  `yaml:"text,omitempty"`
	Key  *string `yaml:"key,omitempty"`
}

// Click represents information needed to click on web components
type Click struct {
	Find string `yaml:"find,omitempty"`
	Wait int64  `yaml:"wait"`
}

// Navigate represents information needed to navigate on defined url
type Navigate struct {
	URL   string `yaml:"url,omitempty"`
	Reset bool   `yaml:"reset,omitempty"`
}

// Select represents information needed to select an option
type Select struct {
	Find string `yaml:"find,omitempty"`
	Text string `yaml:"text,omitempty"`
	Wait int64  `yaml:"wait,omitempty"`
}

// UploadFile represents information needed to upload files
type UploadFile struct {
	Find  string   `yaml:"find,omitempty"`
	Files []string `yaml:"files,omitempty"`
	Wait  int64    `yaml:"wait,omitempty"`
}

// SelectFrame represents information needed to select the frame
type SelectFrame struct {
	Find string `yaml:"find,omitempty"`
}
