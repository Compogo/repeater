# Compogo Repeater 🔁

**Repeater** — это планировщик периодических задач для Compogo, построенный поверх [Runner](https://github.com/Compogo/runner). Поддерживает два режима работы (эксклюзивный и параллельный), имеет собственные middleware и полностью интегрируется с жизненным циклом Compogo.

## 🚀 Установка

```bash
go get github.com/Compogo/repeater
```

### 📦 Быстрый старт

```go
package main

import (
    "context"
    "time"

    "github.com/Compogo/compogo"
    "github.com/Compogo/runner"
    "github.com/Compogo/repeater"
)

func main() {
    app := compogo.NewApp("myapp",
        compogo.WithOsSignalCloser(),
        runner.WithRunner(),
        repeater.WithRepeater(),
        compogo.WithComponents(
            myWorkerComponent,
        ),
    )

    if err := app.Serve(); err != nil {
        panic(err)
    }
}

var myWorkerComponent = &component.Component{
    Dependencies: component.Components{runner.Component, repeater.Component},
    Run: component.StepFunc(func(c container.Container) error {
        return c.Invoke(func(r repeater.Repeater) {
            // Задача, которая выполняется не чаще раза в 5 секунд
            task := repeater.NewTaskWithLock(
                "cleanup",
                runner.ProcessFunc(func(ctx context.Context) error {
                    return doCleanup()
                }),
                5 * time.Second,
            )
            
            r.AddTask(task)
        })
    }),
}
```

### 🎯 Две стратегии выполнения

#### Lock — эксклюзивный режим

```go
// Только один экземпляр задачи может выполняться одновременно
task := repeater.NewTaskWithLock(
    "db-cleanup",
    cleanupProcess,
    1 * time.Hour,
)
```
Если задача ещё работает, следующий запуск пропускается.

#### Unlock — параллельный режим

```go
// Можно запускать сколько угодно экземпляров
task := repeater.NewTaskWithUnlock(
    "queue-worker",
    workerProcess,
    10 * time.Second,
)
```
Каждый запуск получает уникальное имя: queue-worker_1, queue-worker_2 и т.д.

### ⚙️ Конфигурация

Repeater добавляет флаг --repeater.delay — как часто проверять, какие задачи пора запускать.

```bash
./myapp --repeater.delay=100ms
```

По умолчанию — 1/60 секунды (~16.6мс).

### 🧩 Опции задач

#### SkipFirstRun — пропустить первый запуск

```go
task := repeater.NewTaskWithLock(
    "daily-report",
    reportProcess,
    24 * time.Hour,
    repeater.SkipFirstRun,  // первый запуск через 24 часа, а не сразу
)
```

#### 🔌 Middleware

Repeater поддерживает свои middleware, которые работают только с периодическими задачами:

```go
type LoggingMiddleware struct{}

func (m *LoggingMiddleware) Middleware(task *repeater.Task, next runner.Process) runner.Process {
    return runner.ProcessFunc(func(ctx context.Context) error {
        log.Printf("starting periodic task: %s", task.Name())
        err := next.Process(ctx)
        log.Printf("finished periodic task: %s, err=%v", task.Name(), err)
        return err
    })
}

// Использование
r.Use(&LoggingMiddleware{})
```

#### 📊 Мониторинг

У каждой задачи есть счётчик выполненных запусков:

```go
count := task.RunNumbers()  // сколько раз уже запускалась
```

#### 🧹 Graceful shutdown

При остановке приложения Repeater:

* Получает сигнал через Close()
* Завершает основной цикл
* Автоматически останавливает все запущенные экземпляры задач через Runner
