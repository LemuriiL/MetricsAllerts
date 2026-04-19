Использовал pprof дял анализа использования памяти.

снял два профиля памяти:

profiles/base.pprof — до оптимизации
profiles/result.pprof — после оптимизации

Сравнение профилей выполнял командой
go tool pprof -sample_index=alloc_space -top -diff_base="profiles/base.pprof" "profiles/result.pprof"

Результат: 

File: server.exe

Build ID: C:\Users\User\AppData\Local\Temp\go-build98962016\b001\exe\server.exe2026-04-08 21:25:34.7408434 +0300 MSK

Type: alloc_space

Time: 2026-04-08 21:34:01 MSK

Showing nodes accounting for -15MB, 12.34% of 121.53MB total

Dropped 10 nodes (cum <= 0.61MB)

flat  flat%   sum%        cum   cum%

-3.50MB  2.88%  2.88%    -3.50MB  2.88%  net/http.(*Request).WithContext (inline)

3MB  2.47%  0.41%        3MB  2.47%  net/textproto.readMIMEHeader

-3MB  2.47%  2.88%       -3MB  2.47%  github.com/sirupsen/logrus.(*Entry).Dup (inline)

-3MB  2.47%  5.35%       -3MB  2.47%  github.com/sirupsen/logrus.(*Entry).WithFields

3MB  2.47%  2.88%     2.50MB  2.06%  net.(*conn).Read

-2.50MB  2.06%  4.94%    -2.50MB  2.06%  unicode/utf16.Encode

-2MB  1.65%  6.58%       -2MB  1.65%  encoding/json.NewDecoder (inline)

2MB  1.65%  4.94%        4MB  3.29%  net/http.readRequest

-2MB  1.65%  6.58%       -7MB  5.76%  github.com/LemuriiL/MetricsAllerts/internal/server.(*Handler).UpdateMetricJSON

-2MB  1.65%  8.23%       -2MB  1.65%  io.LimitReader (inline)

2MB  1.65%  6.58%   -14.50MB 11.93%  github.com/LemuriiL/MetricsAllerts/internal/server.loggingMiddleware.func1

-1.50MB  1.23%  7.82%     3.50MB  2.88%  net/http.(*conn).readRequest

1.50MB  1.23%  6.58%     1.50MB  1.23%  net/url.parse

-1.50MB  1.23%  7.82%    -1.50MB  1.23%  context.WithValue

-1.50MB  1.23%  9.05%    -1.50MB  1.23%  github.com/LemuriiL/MetricsAllerts/internal/server.gzipMiddleware

-1MB  0.82%  9.88%       -1MB  0.82%  runtime.allocm

-1MB  0.82% 10.70%    -1.50MB  1.23%  encoding/json.(*Decoder).refill

-1MB  0.82% 11.52%       -1MB  0.82%  net/http.Header.Clone (inline)

1MB  0.82% 10.70%        1MB  0.82%  context.withCancel (inline)

-1MB  0.82% 11.52%       -1MB  0.82%  github.com/gorilla/mux.(*Route).Match

-1MB  0.82% 12.35%       -1MB  0.82%  encoding/json.(*decodeState).object

0.50MB  0.41% 11.93%     0.50MB  0.41%  bufio.NewWriterSize (inline)

0.50MB  0.41% 11.52%     0.50MB  0.41%  vendor/golang.org/x/net/http2/hpack.init

-0.50MB  0.41% 11.93%    -0.50MB  0.41%  vendor/golang.org/x/net/dns/dnsmessage.init

0.50MB  0.41% 11.52%     0.50MB  0.41%  sync.(*Pool).pinSlow

0.50MB  0.41% 11.11%     0.50MB  0.41%  runtime.acquireSudog

0.50MB  0.41% 10.70%     0.50MB  0.41%  sync.runtime_notifyListWait

0.50MB  0.41% 10.29%     0.50MB  0.41%  github.com/sirupsen/logrus.(*Logger).releaseEntry

-0.50MB  0.41% 10.70%    -0.50MB  0.41%  time.Time.Format

-0.50MB  0.41% 11.11%    -0.50MB  0.41%  github.com/LemuriiL/MetricsAllerts/internal/server.(*Handler).UpdateMetricsJSON

-0.50MB  0.41% 11.52%    -0.50MB  0.41%  net/textproto.(*Reader).ReadLine (inline)

-0.50MB  0.41% 11.93%     0.50MB  0.41%  context.WithCancel

0.50MB  0.41% 11.52%     0.50MB  0.41%  encoding/json.(*scanner).pushParseState

-0.50MB  0.41% 11.93%       -1MB  0.82%  github.com/sirupsen/logrus.(*TextFormatter).Format

-0.50MB  0.41% 12.34%    -0.50MB  0.41%  internal/syscall/windows.errnoErr (inline)

-0.50MB  0.41% 12.76%    -0.50MB  0.41%  net/http.(*connReader).startBackgroundRead

0.50MB  0.41% 12.34%     0.50MB  0.41%  reflect.New

0     0% 12.34%       -2MB  1.65%  encoding/json.(*Decoder).Decode

0     0% 12.34%       -1MB  0.82%  encoding/json.(*Decoder).readValue

0     0% 12.34%       -1MB  0.82%  encoding/json.(*Encoder).Encode

0     0% 12.34%    -0.50MB  0.41%  encoding/json.(*decodeState).array

0     0% 12.34%       -1MB  0.82%  encoding/json.(*decodeState).unmarshal

0     0% 12.34%       -1MB  0.82%  encoding/json.(*decodeState).value

0     0% 12.34%     0.50MB  0.41%  encoding/json.indirect

0     0% 12.34%     0.50MB  0.41%  encoding/json.stateBeginValue

0     0% 12.34%     0.50MB  0.41%  encoding/json.stateBeginValueOrEmpty

0     0% 12.34%       -1MB  0.82%  github.com/LemuriiL/MetricsAllerts/internal/server.(*loggingResponseWriter).Write

0     0% 12.34%    -7.50MB  6.17%  github.com/LemuriiL/MetricsAllerts/internal/server.gzipMiddleware.func1

0     0% 12.34%    -2.50MB  2.06%  github.com/gorilla/mux.(*Router).Match

0     0% 12.34%      -22MB 18.11%  github.com/gorilla/mux.(*Router).ServeHTTP

0     0% 12.34%    -1.50MB  1.23%  github.com/gorilla/mux.MiddlewareFunc.Middleware

0     0% 12.34%    -1.50MB  1.23%  github.com/gorilla/mux.requestWithRoute

0     0% 12.34%    -3.50MB  2.88%  github.com/gorilla/mux.requestWithVars

0     0% 12.34%    -6.50MB  5.35%  github.com/sirupsen/logrus.(*Entry).Info (inline)

0     0% 12.34%    -6.50MB  5.35%  github.com/sirupsen/logrus.(*Entry).Log

0     0% 12.34%    -6.50MB  5.35%  github.com/sirupsen/logrus.(*Entry).log

0     0% 12.34%    -3.50MB  2.88%  github.com/sirupsen/logrus.(*Entry).write

0     0% 12.34%    -2.50MB  2.06%  github.com/sirupsen/logrus.(*Logger).WithFields

0     0% 12.34%    -2.50MB  2.06%  github.com/sirupsen/logrus.WithFields (inline)

0     0% 12.34%    -0.50MB  0.41%  internal/poll.(*FD).Read

0     0% 12.34%    -2.50MB  2.06%  internal/poll.(*FD).Write

0     0% 12.34%    -2.50MB  2.06%  internal/poll.(*FD).writeConsole

0     0% 12.34%    -0.50MB  0.41%  internal/poll.execIO

0     0% 12.34%    -0.50MB  0.41%  internal/syscall/windows.WSAGetOverlappedResult

0     0% 12.34%     0.50MB  0.41%  io.CopyN

0     0% 12.34%    -0.50MB  0.41%  net.(*netFD).Read

0     0% 12.34%    -0.50MB  0.41%  net/http.(*body).Read

0     0% 12.34%    -0.50MB  0.41%  net/http.(*body).readLocked

0     0% 12.34%     0.50MB  0.41%  net/http.(*chunkWriter).close

0     0% 12.34%     0.50MB  0.41%  net/http.(*chunkWriter).writeHeader

0     0% 12.34%   -17.50MB 14.40%  net/http.(*conn).serve

0     0% 12.34%     0.50MB  0.41%  net/http.(*connReader).abortPendingRead

0     0% 12.34%     2.50MB  2.06%  net/http.(*connReader).backgroundRead

0     0% 12.34%       -1MB  0.82%  net/http.(*response).Write

0     0% 12.34%       -1MB  0.82%  net/http.(*response).WriteHeader

0     0% 12.34%        1MB  0.82%  net/http.(*response).finishRequest

0     0% 12.34%       -1MB  0.82%  net/http.(*response).write

0     0% 12.34%   -14.50MB 11.93%  net/http.HandlerFunc.ServeHTTP

0     0% 12.34%     0.50MB  0.41%  net/http.newBufioWriterSize

0     0% 12.34%     0.50MB  0.41%  net/http.newTextprotoReader

0     0% 12.34%    -2.50MB  2.06%  net/http.readTransfer

0     0% 12.34%      -22MB 18.11%  net/http.serverHandler.ServeHTTP

0     0% 12.34%        3MB  2.47%  net/textproto.(*Reader).ReadMIMEHeader (inline)

0     0% 12.34%     1.50MB  1.23%  net/url.ParseRequestURI

0     0% 12.34%    -2.50MB  2.06%  os.(*File).Write

0     0% 12.34%    -2.50MB  2.06%  os.(*File).write (inline)

0     0% 12.34%     0.50MB  0.41%  runtime.gcMarkDone

0     0% 12.34%       -1MB  0.82%  runtime.mcall

0     0% 12.34%       -1MB  0.82%  runtime.newm

0     0% 12.34%       -1MB  0.82%  runtime.park_m

0     0% 12.34%       -1MB  0.82%  runtime.resetspinning

0     0% 12.34%       -1MB  0.82%  runtime.schedule

0     0% 12.34%     0.50MB  0.41%  runtime.semacquire (inline)

0     0% 12.34%     0.50MB  0.41%  runtime.semacquire1

0     0% 12.34%       -1MB  0.82%  runtime.startm

0     0% 12.34%       -1MB  0.82%  runtime.wakep

0     0% 12.34%     0.50MB  0.41%  sync.(*Cond).Wait

0     0% 12.34%     0.50MB  0.41%  sync.(*Pool).Get

0     0% 12.34%     0.50MB  0.41%  sync.(*Pool).pin
