- [x] Add support for all protocols vmess:// ss:// trojan:// ...
- [x] Add -load for download all and small lists
- [x] Почини проблему с ./xray-checker то что он клонируется отдельным репозиторимем я хочу что бы он шел обычной зависимостью или через vendor
      Сейчас он клонируется и заменяется go.mod через ./Taskfile.yml
      Проблема в том что у него задан в go.mod `module xray-checker`
      Возможно это можно победить через go mod vendor - при это править исходники в ./vendor/ - нельзя
      Возможно тебе поможет //go:linkname или что-то подобное
- [x] Убери именнованные импорты и сделай их обычными
      checkerlogger "github.com/kutovoys/xray-checker/logger"
      checkermodels "github.com/kutovoys/xray-checker/models"
      checkersubscription "github.com/kutovoys/xray-checker/subscription"
      checkerxray "github.com/kutovoys/xray-checker/xray"
- [ ] tag в psauxwwf/xray-checker и kutovoys/xray-checker должен быть синхронизирован сейчас там последний тег v1.3.1
