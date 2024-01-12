package main

import "time"

type Summ struct {
	Size int `json:"size"`
}
type RootEntry struct {
	Summ Summ `json:"summ"`
}

type BackupList []struct {
	ID          string    `json:"id"`
	StartTime   time.Time `json:"startTime"`
	RootEntry   RootEntry `json:"rootEntry"`
	Description string    `json:"description"`
}
type LocalBackupList struct {
	Automatic BackupList `json:"automatic"`
	Ondemand  BackupList `json:"ondemand"`
	System    BackupList `json:"system"`
}

type RemoteBackupList struct {
	Automatic BackupList `json:"automatic"`
	Ondemand  BackupList `json:"ondemand"`
}
type systemstats struct {
	Cores       string `json:"cores"`
	Cpu         string `json:"cpu"`
	TotalMemory string `json:"totalMemory"`
	UsedMemory  string `json:"usedMemory"`
	TotalDisk   string `json:"totalDisk"`
	UsedDisk    string `json:"usedDisk"`
	Bandwidth   string `json:"bandwidth"`
	Os          string `json:"os"`
	Uptime      string `json:"uptime"`
	LoadAvg     string `json:"loadavg"`
	CpuIdeal    string `json:"cpuideal"`
}

type wpadd struct {
	AppName       string `json:"appName"`
	UserName      string `json:"userName"`
	Domain        Domain `json:"domain"`
	Title         string `json:"title"`
	AdminUser     string `json:"adminUser"`
	AdminPassword string `json:"adminPassword"`
	AdminEmail    string `json:"adminEmail"`
	Routing       string `json:"routing"`
}

type db struct {
	Name     string `json:"name"`
	User     string `json:"user"`
	Password string `json:"password"`
}

type wpdelete struct {
	Main struct {
		Name string `json:"name"`
		User string `json:"user"`
	} `json:"main"`
	Staging struct {
		Name string `json:"name"`
		User string `json:"user"`
	} `json:"staging"`
	IsStaging bool `json:"isStaging"`
}

// type wpcert struct {
// 	AppName     string  `json:"appName"`
// 	Url         string  `json:"url"`
// 	Type        string  `json:"domainType"`
// 	IsSubdomain string  `json:"isSubdomain"`
// 	SslConf     sslConf `json:"sslConf"`
// }

// type sslConf struct {
// 	IsWildcard string `json:"isWildcard"`
// 	Type       string `json:"type"`
// 	KeyType    string `json:"keyType"`
// 	CustomKey  string `json:"customKey"`
// 	CustomCert string `json:"CustomCert"`
// }

type sslConf struct {
	App       string `json:"app"`
	User      string `json:"user"`
	Challenge string `json:"challenge"`
	Custom    struct {
		Certificate string `json:"certificate"`
		Key         string `json:"key"`
	} `json:"custom"`
	Domain      string   `json:"domainName"`
	Domains     []string `json:"domains"`
	Provider    string   `json:"provider"`
	DNSProvider string   `json:"dnsProvider"`
	Token       string   `json:"token"`
}

type errcode struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type errJson struct {
	Errors []struct {
		Field   string `json:"field"`
		Message string `json:"message"`
	} `json:"errors"`
}

// type DomainEdit struct {
// 	Name string `json:"name"`
// 	Site Site   `json:"site"`
// }

type Domain struct {
	Url       string `json:"url"`
	SubDomain bool   `json:"subDomain"`
	Routing   string `json:"routing"`
	Wildcard  bool   `json:"wildcard"`
	Type      int    `json:"type"`
}

type DomainConf struct {
	Domain   Domain `json:"domain" validate:"required"`
	SiteName string `json:"site" validate:"required"`
}

type PrimaryChange struct {
	Name           string `json:"name"`
	CurrentPrimary string `json:"currentPrimary"`
	NewPrimary     string `json:"newPrimary"`
	User           string `json:"user"`
}

type PHPChange struct {
	Name string `json:"name"`
	// Sites  []Site `json:"sites"`
	OldPHP string `json:"oldphp"`
	NewPHP string `json:"newphp"`
}

type PHP struct {
	MaxExecutionTime      string `ini:"max_execution_time"`
	MaxFileUploads        string `ini:"max_file_uploads"`
	MaxInputTime          string `ini:"max_input_time"`
	MaxInputVars          string `ini:"max_input_vars"`
	MemoryLimit           string `ini:"memory_limit"`
	PostMaxSize           string `ini:"post_max_size"`
	SessionCookieLifetime string `ini:"session.cookie_lifetime"`
	SessionGcMaxlifetime  string `ini:"session.gc_maxlifetime"`
	ShortOpenTag          string `ini:"short_open_tag"`
	UploadMaxFilesize     string `ini:"upload_max_filesize"`
	Timezone              string `ini:"date.timezone"`
	OpenBaseDir           string `ini:"open_basedir"`
}

type PhpIniParsed struct {
	MaxExecutionTime      int    `ini:"max_execution_time"`
	MaxFileUploads        int    `ini:"max_file_uploads"`
	MaxInputTime          int    `ini:"max_input_time"`
	MaxInputVars          int    `ini:"max_input_vars"`
	MemoryLimit           int    `ini:"memory_limit"`
	PostMaxSize           int    `ini:"post_max_size"`
	SessionCookieLifetime int    `ini:"session.cookie_lifetime"`
	SessionGcMaxlifetime  int    `ini:"session.gc_maxlifetime"`
	ShortOpenTag          string `ini:"short_open_tag"`
	UploadMaxFilesize     int    `ini:"upload_max_filesize"`
	Timezone              string `ini:"date.timezone"`
	OpenBaseDir           string `ini:"open_basedir"`
}

type PHPini struct {
	PHP
}

//////////////////////////////////////////////////////////////////////////////////////////////////////

type Global struct {
	Datasize int `json:"dataSize"`
	Maxconn  int `json:"maxConnection"`
}
type Timeout struct {
	Connect int `json:"connect"`
	Client  int `json:"client"`
	Server  int `json:"server"`
}

type Default struct {
	Timeout Timeout `json:"timeout"`
}

type Site struct {
	Name         string         `json:"name"`
	User         string         `json:"user"`
	Domains      []string       `json:"domains"`
	Cache        string         `json:"cache"`
	LocalBackup  Backup         `json:"localBackup"`
	RemoteBackup []RemoteBackup `json:"remoteBackup"`
	Type         string         `json:"type"`
	EnforceHttps bool           `json:"enforceHttps"`
}

// type NewRelic struct {
// 	Status  string `json:"status"`
// 	AppName string `json:"appname"`
// 	Key     string `json:"key"`
// }

type Config struct {
	Sites []Site `json:"sites"`
}

type DomainJSON struct {
	Url string `json:"url"`
	SSL struct {
		FolderName string `json:"folderName"`
	} `json:"ssl"`
	WildCard bool `json:"wildcard"`
}

type Backup struct {
	Automatic bool            `json:"automatic"`
	Frequency string          `json:"frequency"`
	Time      BackupTime      `json:"time"`
	Retention BackupRetention `json:"retention"`
	LastRun   string          `json:"lastrun"`
}

type RemoteBackup struct {
	Provider string `json:"provider"`
	Bucket   string `json:"bucket"`
	Backup
}

type BackupTime struct {
	Hour     string `json:"hour"`
	Minute   string `json:"minute"`
	MonthDay string `json:"monthday"`
	WeekDay  string `json:"weekday"`
}

type BackupRetention struct {
	Type string `json:"type"`
	Time int    `json:"time"`
}

type Staging struct {
	Name        string `json:"name"`
	User        string `json:"user"`
	Type        string `json:"type"`
	Url         string `json:"url"`
	LivesiteUrl string `json:"livesiteurl"`
}

type SyncChanges struct {
	// Method string   `json:"method"`
	Type []string `json:"type"`
	From struct {
		Name string `json:"name"`
		User string `json:"user"`
		Type string `json:"type"`
		Url  string `json:"url"`
	} `json:"fromSite"`
	To struct {
		Name string `json:"name"`
		User string `json:"user"`
		Type string `json:"type"`
		Url  string `json:"url"`
	} `json:"toSite"`
	DbType      string   `json:"dbType"`
	AllSelected bool     `json:"allSelected"`
	Tables      []string `json:"tables"`
	CopyMethod  string   `json:"copyMethod"`
	Exclude     struct {
		IsExclude bool     `json:"isexclude"`
		Files     []string `json:"files"`
		Folders   []string `json:"folders"`
	} `json:"exclude"`
	DeleteDestFiles bool `json:"deleteDestFiles"`
}

type SSH struct {
	Key       string `json:"key"`
	User      string `json:"user"`
	Label     string `json:"label"`
	Timestamp int    `json:"timestamp"`
}

type PluginsThemesOperation struct {
	Plugins []struct {
		Name      string `json:"name"`
		Operation string `json:"operation"`
	}
	Themes []struct {
		Name      string `json:"name"`
		Operation string `json:"operation"`
	}
}

type EnforceHttps struct {
	Operation string `json:"operation"`
	Name      string `json:"name"`
}
type MetricsValue struct {
	Time  int     `json:"time"`
	Value float64 `json:"value"`
}

type SingleService struct {
	Service string `json:"service"`
	Running bool   `json:"running"`
	Process string `json:"process"`
}

type FieldError struct {
	Error struct {
		Field   string `json:"field"`
		Message string `json:"message"`
	} `json:"error"`
}

type Clone struct {
	Original struct {
		Name   string `json:"name"`
		User   string `json:"user"`
		Domain string `json:"domain"`
	} `json:"originalSite"`
	Clone struct {
		Name   string `json:"name"`
		User   string `json:"user"`
		Domain Domain `json:"domain"`
	} `json:"cloneSite"`
	Rewrite bool `json:"rewrite"`
}

type AuthorizedKey struct {
	Username  string
	Key       string
	Label     string
	Timestamp int64
}

type ServiceAction struct {
	Action  string `json:"action"`
	Service string `json:"service"`
}

type UpdateWildcardResp struct {
	Domain struct {
		Url       string `json:"url" validation:"required"`
		Wildcard  bool   `json:"wildcard" validation:"required"`
		Subdomain bool   `json:"subdomain" validation:"required"`
	}
	Site string `json:"site"`
}

type DeleteDomain struct {
	Domain string `json:"domain" validation:"required"`
	Site   string `json:"site" validation:"required"`
}

type Error struct {
	Message string `json:"message"`
}
