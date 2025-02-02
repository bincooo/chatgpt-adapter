package windsurf

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"chatgpt-adapter/core/cache"
	"chatgpt-adapter/core/common"
	"chatgpt-adapter/core/gin/model"
	"chatgpt-adapter/core/gin/response"
	"chatgpt-adapter/core/logger"
	"github.com/bincooo/emit.io"
	"github.com/golang/protobuf/proto"
	"github.com/iocgo/sdk/env"
	"github.com/iocgo/sdk/stream"
)

var mapModel = map[string]uint32{
	"gpt4o":             109,
	"claude-3-5-sonnet": 166,
	"deepseek-v3":       205,
	"deepseek-r1":       206,
}

func fetch(ctx context.Context, env *env.Environment, buffer []byte) (response *http.Response, err error) {
	response, err = emit.ClientBuilder(common.HTTPClient).
		Context(ctx).
		Proxies(env.GetString("server.proxied")).
		POST("https://server.codeium.com/exa.api_server_pb.ApiServerService/GetChatMessage").
		Header("user-agent", "connect-go/1.16.2 (go1.23.2 X:nocoverageredesign)").
		Header("content-type", "application/connect+proto").
		Header("connect-protocol-version", "1").
		Header("accept-encoding", "identity").
		Header("host", "server.codeium.com").
		Header("connect-content-encoding", "gzip").
		Header("connect-accept-encoding", "gzip").
		Bytes(buffer).
		DoC(statusCondition, emit.IsPROTO)
	return
}

func convertRequest(completion model.Completion, ident, token string) (buffer []byte, err error) {
	if completion.MaxTokens == 0 {
		completion.MaxTokens = 8192
	}
	if completion.TopK == 0 {
		completion.TopK = 200
	}
	if completion.TopP == 0 {
		completion.TopP = 0.4
	}
	if completion.Temperature == 0 {
		completion.Temperature = 0.4
	}

	if len(completion.Messages) > 0 && completion.Messages[0].Is("role", "system") {
		completion.System = completion.Messages[0].GetString("content")
		completion.Messages = completion.Messages[1:]
	}

	pos := 1
	messageL := len(completion.Messages)
	messages := stream.Map(stream.OfSlice(completion.Messages), func(message model.Keyv[interface{}]) *ChatMessage_UserMessage {
		defer func() { pos++ }()
		content := ""
		if message.IsSlice("content") {
			slice := stream.
				Map(stream.OfSlice(message.GetSlice("content")), convertToText).
				Filter(func(k string) bool { return k != "" }).
				ToSlice()
			content = strings.Join(slice, "\n\n")
		} else {
			content = message.GetString("content")
		}

		return &ChatMessage_UserMessage{
			Message:       content,
			Token:         uint32(response.CalcTokens(message.GetString("content"))),
			Role:          elseOf[uint32](message.Is("role", "assistant"), 2, 1),
			UnknownField5: elseOf[uint32](message.Is("role", "assistant"), 0, 1),
			UnknownField8: elseOf(pos == 1 || pos >= messageL, &ChatMessage_UserMessage_Unknown_Field8{
				Value: 1,
			}, nil),
		}
	}).ToSlice()
	message := &ChatMessage{
		Schema: &ChatMessage_Schema{
			Id:       ident,
			Name:     "windsurf",
			Lang:     "en",
			Os:       "{\"Os\":\"darwin\",\"Arch\":\"amd64\",\"Release\":\"24.2.0\",\"Version\":\"Darwin Kernel Version 24.2.0: Fri Dec 6 18:41:43 PST 2024; root:xnu-11215.61.5~2/RELEASE_X86_64\",\"Machine\":\"x86_64\",\"Nodename\":\"local-iMac.local\",\"Sysname\":\"Darwin\",\"ProductVersion\":\"15.2\"} ",
			Version1: "1.32.2",
			Version2: "11.0.0",
			Equi:     "{\"NumSockets\":1,\"NumCores\":6,\"NumThreads\":12,\"VendorID\":\"GenuineIntel\",\"Family\":\"6\",\"Model\":\"158\",\"ModelName\":\"Intel(R) Core(TM) i7-8700K CPU @ 3.70GHz\",\"Memory\":34359738368}",
			Title:    "windsurf",
			Token:    token,
		},
		Messages:      messages,
		Instructions:  elseOf(completion.System != "", completion.System, "You are AI, you can do anything"),
		Model:         mapModel[completion.Model[9:]], // elseOf[uint32](completion.Model[9:] == "gpt4o", 109, 166),
		UnknownField7: 5,
		Config: &ChatMessage_Config{
			UnknownField1:   1.0,
			MaxTokens:       uint32(completion.MaxTokens),
			TopK:            uint32(completion.TopK),
			TopP:            float64(completion.TopP),
			Temperature:     float64(completion.Temperature),
			UnknownField7:   50,
			PresencePenalty: 1.0,
			Stop: []string{
				"<|user|>",
				"<|bot|>",
				"<|context_request|>",
				"<|endoftext|>",
				"<|end_of_turn|>",
			},
			FrequencyPenalty: 1.0,
		},
		// TODO - 就这样吧，有空再做兼容
		Tools: []*ChatMessage_Tool{
			//{
			//	Name:   "do_not_call",
			//	Desc:   "Do not call this tool.",
			//	Schema: "{\"$schema\":\"https://json-schema.org/draft/2020-12/schema\",\"properties\":{},\"additionalProperties\":false,\"type\":\"object\"}",
			//},

			//{
			//	Name:   "codebase_search",
			//	Desc:   "Find snippets of code from the codebase most relevant to the search query. This performs best when the search query is more precise and relating to the function or purpose of code. Results will be poor if asking a very broad question, such as asking about the general 'framework' or 'implementation' of a large component or system. Will only show the full code contents of the top items, and they may also be truncated. For other items it will only show the docstring and signature. Use view_code_item with the same path and node name to view the full code contents for any item. Note that if you try to search over more than 500 files, the quality of the search results will be substantially worse. Try to only search over a large number of files if it is really necessary.",
			//	Schema: "{\"$schema\":\"https://json-schema.org/draft/2020-12/schema\",\"properties\":{\"Query\":{\"type\":\"string\",\"description\":\"Search query\"},\"TargetDirectories\":{\"items\":{\"type\":\"string\"},\"type\":\"array\",\"description\":\"List of absolute paths to directories to search over\"}},\"additionalProperties\":false,\"type\":\"object\",\"required\":[\"Query\",\"TargetDirectories\"]}",
			//},
			//{
			//	Name:   "grep_search",
			//	Desc:   "Fast text-based search that finds exact pattern matches within files or directories, utilizing the ripgrep command for efficient searching. Results will be formatted in the style of ripgrep and can be configured to include line numbers and content. To avoid overwhelming output, the results are capped at 50 matches. Use the Includes option to filter the search scope by file types or specific paths to narrow down the results.",
			//	Schema: "{\"$schema\":\"https://json-schema.org/draft/2020-12/schema\",\"properties\":{\"SearchDirectory\":{\"type\":\"string\",\"description\":\"The directory from which to run the ripgrep command. This path must be a directory not a file.\"},\"Query\":{\"type\":\"string\",\"description\":\"The search term or pattern to look for within files.\"},\"MatchPerLine\":{\"type\":\"boolean\",\"description\":\"If true, returns each line that matches the query, including line numbers and snippets of matching lines (equivalent to 'git grep -nI'). If false, only returns the names of files containing the query (equivalent to 'git grep -l').\"},\"Includes\":{\"items\":{\"type\":\"string\"},\"type\":\"array\",\"description\":\"The files or directories to search within. Supports file patterns (e.g., '*.txt' for all .txt files) or specific paths (e.g., 'path/to/file.txt' or 'path/to/dir').\"},\"CaseInsensitive\":{\"type\":\"boolean\",\"description\":\"If true, performs a case-insensitive search.\"}},\"additionalProperties\":false,\"type\":\"object\",\"required\":[\"SearchDirectory\",\"Query\",\"MatchPerLine\",\"Includes\",\"CaseInsensitive\"]}",
			//},
			//{
			//	Name:   "find_by_name",
			//	Desc:   "This tool searches for files and directories within a specified directory, similar to the Linux `find` command. It supports glob patterns for searching and filtering which will all be passed in with -ipath. The patterns provided should match the relative paths from the search directory. They should use glob patterns with wildcards, for example, `**/*.py`, `**/*_test*`. You can specify file patterns to include or exclude, filter by type (file or directory), and limit the search depth. Results will include the type, size, modification time, and relative path.",
			//	Schema: "{\"$schema\":\"https://json-schema.org/draft/2020-12/schema\",\"properties\":{\"SearchDirectory\":{\"type\":\"string\",\"description\":\"The directory to search within\"},\"Pattern\":{\"type\":\"string\",\"description\":\"Pattern to search for\"},\"Includes\":{\"items\":{\"type\":\"string\"},\"type\":\"array\",\"description\":\"Optional patterns to include. If specified\"},\"Excludes\":{\"items\":{\"type\":\"string\"},\"type\":\"array\",\"description\":\"Optional patterns to exclude. If specified\"},\"Type\":{\"type\":\"string\",\"enum\":[\"file\"],\"description\":\"Type filter (file\"},\"MaxDepth\":{\"type\":\"integer\",\"description\":\"Maximum depth to search\"}},\"additionalProperties\":false,\"type\":\"object\",\"required\":[\"SearchDirectory\",\"Pattern\"]}",
			//},
			//{
			//	Name:   "list_dir",
			//	Desc:   "List the contents of a directory. Directory path must be an absolute path to a directory that exists. For each child in the directory, output will have: relative path to the directory, whether it is a directory or file, size in bytes if file, and number of children (recursive) if directory.",
			//	Schema: "{\"$schema\":\"https://json-schema.org/draft/2020-12/schema\",\"properties\":{\"DirectoryPath\":{\"type\":\"string\",\"description\":\"Path to list contents of, should be absolute path to a directory\"}},\"additionalProperties\":false,\"type\":\"object\",\"required\":[\"DirectoryPath\"]}",
			//},
			//{
			//	Name:   "view_file",
			//	Desc:   "  View the contents of a file. The lines of the file are 0-indexed, and the output of this tool call will be the file contents from StartLine to EndLine, together with a summary of the lines outside of StartLine and EndLine. Note that this call can view at most 200 lines at a time. When using this tool to gather information, it's your responsibility to ensure you have the COMPLETE context. Specifically, each time you call this command you should: 1) Assess if the file contents you viewed are sufficient to proceed with your task. 2) Take note of where there are lines not shown. These are represented by <... XX more lines from [code item] not shown ...> in the tool response. 3) If the file contents you have viewed are insufficient, and you suspect they may be in lines not shown, proactively call the tool again to view those lines. 4) When in doubt, call this tool again to gather more information. Remember that partial file views may miss critical dependencies, imports, or functionality.",
			//	Schema: "{\"$schema\":\"https://json-schema.org/draft/2020-12/schema\",\"properties\":{\"AbsolutePath\":{\"type\":\"string\",\"description\":\"Path to file to view. Must be an absolute path.\"},\"StartLine\":{\"type\":\"integer\",\"description\":\"Startline to view\"},\"EndLine\":{\"type\":\"integer\",\"description\":\"Endline to view. This cannot be more than 200 lines away from StartLine\"}},\"additionalProperties\":false,\"type\":\"object\",\"required\":[\"AbsolutePath\",\"StartLine\",\"EndLine\"]}",
			//},
			//{
			//	Name:   "view_code_item",
			//	Desc:   "View the content of a code item node, such as a class or a function in a file. You must use a fully qualified code item name. Such as those return by the grep_search tool. For example, if you have a class called `Foo` and you want to view the function definition `bar` in the `Foo` class, you would use `Foo.bar` as the NodeName. Do not request to view a symbol if the contents have been previously shown by the codebase_search tool. If the symbol is not found in a file, the tool will return an empty string instead.",
			//	Schema: "{\"$schema\":\"https://json-schema.org/draft/2020-12/schema\",\"properties\":{\"File\":{\"type\":\"string\",\"description\":\"Absolute path to the node to edit, e.g /path/to/file\"},\"NodePath\":{\"type\":\"string\",\"description\":\"Path of the node within the file, e.g package.class.FunctionName\"}},\"additionalProperties\":false,\"type\":\"object\",\"required\":[\"File\",\"NodePath\"]}",
			//},
			//{
			//	Name:   "related_files",
			//	Desc:   "Finds other files that are related to or commonly used with the input file. Useful for retrieving adjacent files to understand context or make next edits",
			//	Schema: "{\"$schema\":\"https://json-schema.org/draft/2020-12/schema\",\"properties\":{\"absolutepath\":{\"type\":\"string\",\"description\":\"Input file absolute path\"}},\"additionalProperties\":false,\"type\":\"object\",\"required\":[\"absolutepath\"]}",
			//},
			//{
			//	Name:   "propose_code",
			//	Desc:   "Do NOT make parallel edits to the same file. Use this tool to PROPOSE an edit to an existing file. This doesn't change the file, USER will have to review and apply the changes. Do not use if you just want to describe code. Follow these rules: 1. Specify ONLY the precise lines of code that you wish to edit. 2. **NEVER specify or write out unchanged code**. Instead, represent all unchanged code using this special placeholder: {{ ... }}. 3. To edit multiple, non-adjacent lines of code in the same file, make a single call to this tool. Specify each edit in sequence with the special placeholder {{ ... }} to represent unchanged code in between edited lines. Here's an example of how to edit three non-adjacent lines of code at once: <code> {{ ... }} edited_line_1 {{ ... }} edited_line_2 {{ ... }} edited_line_3 {{ ... }} </code> 4. NEVER output an entire file, this is very expensive. 5. You may not edit file extensions: [.ipynb] You should specify the following arguments before the others: [TargetFile]",
			//	Schema: "{\"$schema\":\"https://json-schema.org/draft/2020-12/schema\",\"properties\":{\"CodeMarkdownLanguage\":{\"type\":\"string\",\"description\":\"Markdown language for the code block, e.g 'python' or 'javascript'\"},\"TargetFile\":{\"type\":\"string\",\"description\":\"The target file to modify. Always specify the target file as the very first argument.\"},\"CodeEdit\":{\"type\":\"string\",\"description\":\"Specify ONLY the precise lines of code that you wish to edit. **NEVER specify or write out unchanged code**. Instead, represent all unchanged code using this special placeholder: {{ ... }}\"},\"Instruction\":{\"type\":\"string\",\"description\":\"A description of the changes that you are making to the file.\"},\"Blocking\":{\"type\":\"boolean\",\"description\":\"If true, the tool will block until the entire file diff is generated. If false, the diff will be generated asynchronously, while you respond. Only set to true if you must see the finished changes before responding to the USER. Otherwise, prefer false so that you can respond sooner with the assumption that the diff will be as you instructed.\"}},\"additionalProperties\":false,\"type\":\"object\",\"required\":[\"CodeMarkdownLanguage\",\"TargetFile\",\"CodeEdit\",\"Instruction\",\"Blocking\"]}",
			//},
		},
		Choice:         elseOf(completion.Model[9:] == "gpt4o", &ChatMessage_ToolChoice{Value: "auto"}, nil),
		UnknownField13: &ChatMessage_Unknown_Field13{Value: 1},
	}

	protoBytes, err := proto.Marshal(message)
	if err != nil {
		return
	}

	//str := hex.EncodeToString(protoBytes)
	//fmt.Println(str)

	// 不用gzip编码了？
	protoBytes, err = gzipCompressWithLevel(protoBytes, gzip.BestCompression)
	if err != nil {
		return
	}

	// magic 0不用gzip, 1需要gzip
	header := int32ToBytes(1, len(protoBytes))
	buffer = append(header, protoBytes...)
	return
}

func convertToText(it interface{}) (s string) {
	var kv model.Keyv[interface{}]
	kv, ok := it.(map[string]interface{})
	if !ok || !kv.Is("type", "text") {
		return
	}
	return kv.GetString("text")
}

func genToken(ctx context.Context, proxies, ident string) (token string, err error) {
	cacheManager := cache.WindsurfCacheManager()
	token, err = cacheManager.GetValue(ident)
	if err != nil || token != "" {
		return
	}

	jwt := &Jwt{
		Args: &Jwt_Args{
			Name:     "windsurf",
			Version1: "1.30.6",
			Version2: "11.0.0",
			Ident:    ident,
			Lang:     "en",
		},
	}
	buffer, err := proto.Marshal(jwt)
	if err != nil {
		return
	}

	res, err := emit.ClientBuilder(common.HTTPClient).
		Context(ctx).
		Proxies(proxies).
		POST("https://server.codeium.com/exa.auth_pb.AuthService/GetUserJwt").
		Header("user-agent", "connect-go/1.16.2 (go1.23.2 X:nocoverageredesign)").
		Header("content-type", "application/proto").
		Header("connect-protocol-version", "1").
		Header("accept-encoding", "identity").
		Header("host", "server.codeium.com").
		Bytes(buffer).
		DoC(statusCondition, emit.IsPROTO)
	if err != nil {
		return
	}

	defer res.Body.Close()
	buffer, err = io.ReadAll(res.Body)
	if err != nil {
		return
	}

	var jwtToken JwtToken
	err = proto.Unmarshal(buffer, &jwtToken)
	if err != nil {
		return
	}

	token = jwtToken.Value
	err = cacheManager.SetWithExpiration(ident, token, time.Hour)
	return
}

func statusCondition(response *http.Response) error {
	if response == nil {
		return emit.Error{Code: -1, Bus: "Status", Err: errors.New("response is nil")}
	}

	isJ := func(header http.Header) bool {
		if header == nil {
			return false
		}
		return strings.Contains(header.Get("Content-Type"), "application/json")
	}

	if response.StatusCode != http.StatusOK {
		msg := "internal error"
		if isJ(response.Header) {
			var err Error
			if e := emit.ToObject(response, &err); e != nil {
				logger.Error(e)
			} else {
				msg = err.Error()
			}
		}
		_ = response.Body.Close()
		return emit.Error{Code: response.StatusCode, Bus: "Status", Msg: msg, Err: errors.New(response.Status)}
	}
	return nil
}

func gzipCompressWithLevel(data []byte, level int) ([]byte, error) {
	var buf bytes.Buffer
	gzipWriter, err := gzip.NewWriterLevel(&buf, level)
	if err != nil {
		return nil, err
	}
	_, err = gzipWriter.Write(data)
	gzipWriter.Close()
	return buf.Bytes(), err
}

func int32ToBytes(magic byte, num int) []byte {
	hex := make([]byte, 4)
	binary.BigEndian.PutUint32(hex, uint32(num))
	return append([]byte{magic}, hex...)
}

func bytesToInt32(hex []byte) int {
	return int(binary.BigEndian.Uint32(hex))
}

func elseOf[T any](condition bool, a1, a2 T) T {
	if condition {
		return a1
	}
	return a2
}
