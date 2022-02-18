# rkndaemon

Программа для скачивания реестра запрещенных сайтов и построения списков url, domain, domain-mask ip, subnets файлов.
Поддерживает DNS резолвинг доменов для дальнейшей блокировки их по IP.

Запускается как демон, постоянно висит в памяти и проверяет наличие обновлений в РКН.
Для доступа к выгрузке нужен логин/пароль от сайта РКН, необходимо в личном кабинете указать IP адрес с которого будет выгрузка.

## выгрузка социально значимых сайтов

в директории output

`SocDomais.txt`

`SocNets.txt`



### установка:

`go install github.com/prgra/rkndaemon@latest`


### запуск:

```bash
RKN_USER=4433221100 RKN_PASS=password rkndaemon
```

конфигурационный файл `rkndaemon.toml` должен находиться в директории с программой либо в `/etc/rkndaemon.toml`

настройки по умолчанию:

```toml
	rknurl = "http://vigruzki2.rkn.gov.ru/services/OperatorRequest2/?wsdl"
	rknuser = ""
	rknpass = ""
	dnses = ["8.8.8.8", "1.1.1.1"]
	dnsworkers = 64
	resolvfile = "output/resolved.txt"
	socinterval = 60
	dumpinterval = 5
	postscript = ""
	socialscript = ""
	usedump = true
	usesoc = true
	useresolver = false
```

конфигурирование через переменные окружения env, имеют больший приоритет чем конфигурационный файл

```
	RKN_URL
	RKN_USER
	RKN_PASS
	RKN_DNSSERVERS
	RKN_WORKERCOUNT
	RKN_RESOLVERFILE
	RKN_SOCIALINTERVAL
	RKN_DUMPINTERVAL
	RKN_POSTSCRIPT
	RKN_SOCIALSCRIPT
	RKN_USEDUMP
	RKN_USESOC
	RKN_USERESOLVER
```
обработанные файлы складываются в директорию `output`