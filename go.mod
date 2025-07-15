module chatgpt-adapter

go 1.23.3

require (
	github.com/antonfisher/nested-logrus-formatter v1.3.1
	github.com/bincooo/coze-api v1.0.2-0.20250118010946-7c4f3c5e25ea
	github.com/bincooo/edge-api v1.0.4-0.20250211074233-37fe84649a9b
	github.com/bincooo/emit.io v1.0.1-0.20250327152715-789fc5920a10
	github.com/bincooo/you.com v0.0.0-20250205070606-666b6847729b
	github.com/bogdanfinn/tls-client v1.8.0
	github.com/dlclark/regexp2 v1.11.4
	github.com/eko/gocache/lib/v4 v4.1.6
	github.com/eko/gocache/store/go_cache/v4 v4.2.2
	github.com/gabriel-vasile/mimetype v1.4.3
	github.com/gin-gonic/gin v1.10.0
	github.com/golang-jwt/jwt/v5 v5.2.2
	github.com/golang/protobuf v1.5.4
	github.com/google/uuid v1.6.0
	github.com/iocgo/sdk v0.0.0-20241203133330-43dcedf3291e
	github.com/lestrrat-go/file-rotatelogs v2.4.0+incompatible
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/samber/go-gpt-3-encoder v0.3.1
	github.com/sirupsen/logrus v1.9.3
	github.com/wasmerio/wasmer-go v1.0.5-0.20250109124841-f09913d8a0be
	google.golang.org/protobuf v1.36.0
)

//github.com/iocgo/sdk v0.0.0-20241129021727-ca323c08f298 => ../sdk
//github.com/bincooo/edge-api v1.0.4-0.20250107025218-74fbeaa104b8 => ../edge-api
replace github.com/samber/do/v2 v2.0.0-beta.7 => github.com/iocgo/do/v2 v2.0.0-patch.0.20241204032939-7bbcadbc5f38

require (
	github.com/RomiChan/websocket v1.4.3-0.20220227141055-9b2c6168c9c5 // indirect
	github.com/andybalholm/brotli v1.1.1 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bincooo/go-annotation v0.0.0-20250715054007-ed92d574bb99 // indirect
	github.com/bogdanfinn/fhttp v0.5.36 // indirect
	github.com/bogdanfinn/utls v1.6.5 // indirect
	github.com/bytedance/sonic v1.11.6 // indirect
	github.com/bytedance/sonic/loader v0.1.1 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/cloudflare/circl v1.5.0 // indirect
	github.com/cloudwego/base64x v0.1.4 // indirect
	github.com/cloudwego/iasm v0.2.0 // indirect
	github.com/fsnotify/fsnotify v1.7.0 // indirect
	github.com/gin-contrib/sse v0.1.0 // indirect
	github.com/gingfrederik/docx v0.0.1 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-playground/validator/v10 v10.20.0 // indirect
	github.com/goccy/go-json v0.10.2 // indirect
	github.com/golang/mock v1.6.0 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jonboulle/clockwork v0.4.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/compress v1.17.11 // indirect
	github.com/klauspost/cpuid/v2 v2.2.7 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/lestrrat-go/strftime v1.1.0 // indirect
	github.com/magiconair/properties v1.8.7 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/pelletier/go-toml/v2 v2.2.2 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/prometheus/client_golang v1.19.1 // indirect
	github.com/prometheus/client_model v0.5.0 // indirect
	github.com/prometheus/common v0.48.0 // indirect
	github.com/prometheus/procfs v0.12.0 // indirect
	github.com/quic-go/quic-go v0.48.1 // indirect
	github.com/sagikazarmark/locafero v0.4.0 // indirect
	github.com/sagikazarmark/slog-shim v0.1.0 // indirect
	github.com/samber/do/v2 v2.0.0-beta.7 // indirect
	github.com/samber/go-type-to-string v1.6.1 // indirect
	github.com/samber/lo v1.37.0 // indirect
	github.com/sourcegraph/conc v0.3.0 // indirect
	github.com/spf13/afero v1.11.0 // indirect
	github.com/spf13/cast v1.6.0 // indirect
	github.com/spf13/cobra v1.8.1 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/spf13/viper v1.19.0 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/tam7t/hpkp v0.0.0-20160821193359-2b70b4024ed5 // indirect
	github.com/tidwall/gjson v1.18.0 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.0 // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	github.com/ugorji/go/codec v1.2.12 // indirect
	go.uber.org/atomic v1.9.0 // indirect
	go.uber.org/multierr v1.9.0 // indirect
	golang.org/x/arch v0.8.0 // indirect
	golang.org/x/crypto v0.36.0 // indirect
	golang.org/x/exp v0.0.0-20240719175910-8a7402abbf56 // indirect
	golang.org/x/mod v0.21.0 // indirect
	golang.org/x/net v0.37.0 // indirect
	golang.org/x/sync v0.12.0 // indirect
	golang.org/x/sys v0.31.0 // indirect
	golang.org/x/text v0.23.0 // indirect
	golang.org/x/tools v0.25.0 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
