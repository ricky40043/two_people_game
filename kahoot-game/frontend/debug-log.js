// 前端調試日誌系統
class DebugLogger {
  constructor() {
    this.logs = []
    this.maxLogs = 1000
  }

  log(level, category, message, data = null) {
    const timestamp = new Date().toISOString()
    const logEntry = {
      timestamp,
      level,
      category,
      message,
      data: data ? JSON.stringify(data, null, 2) : null
    }
    
    this.logs.push(logEntry)
    
    // 保持日誌數量限制
    if (this.logs.length > this.maxLogs) {
      this.logs.shift()
    }
    
    // 同時輸出到控制台
    const consoleMessage = `[${timestamp}] [${level}] [${category}] ${message}`
    
    if (data) {
      console.log(consoleMessage, data)
    } else {
      console.log(consoleMessage)
    }
    
    // 更新頁面上的日誌顯示
    this.updateLogDisplay()
  }

  info(category, message, data) {
    this.log('INFO', category, message, data)
  }

  warn(category, message, data) {
    this.log('WARN', category, message, data)
  }

  error(category, message, data) {
    this.log('ERROR', category, message, data)
  }

  debug(category, message, data) {
    this.log('DEBUG', category, message, data)
  }

  updateLogDisplay() {
    // 如果頁面上有日誌顯示區域，更新它
    const logDisplay = document.getElementById('debug-log-display')
    if (logDisplay) {
      const recentLogs = this.logs.slice(-50) // 只顯示最近50條
      logDisplay.innerHTML = recentLogs
        .map(log => {
          const dataStr = log.data ? `\n${log.data}` : ''
          return `<div class="log-entry log-${log.level.toLowerCase()}">
            <span class="log-time">${log.timestamp.split('T')[1].split('.')[0]}</span>
            <span class="log-category">[${log.category}]</span>
            <span class="log-message">${log.message}</span>
            ${dataStr ? `<pre class="log-data">${dataStr}</pre>` : ''}
          </div>`
        })
        .join('')
      
      // 自動滾動到底部
      logDisplay.scrollTop = logDisplay.scrollHeight
    }
  }

  exportLogs() {
    const logsText = this.logs
      .map(log => {
        const dataStr = log.data ? `\nData: ${log.data}` : ''
        return `${log.timestamp} [${log.level}] [${log.category}] ${log.message}${dataStr}`
      })
      .join('\n\n')
    
    const blob = new Blob([logsText], { type: 'text/plain' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `debug-log-${new Date().toISOString().split('T')[0]}.txt`
    a.click()
    URL.revokeObjectURL(url)
  }

  clear() {
    this.logs = []
    this.updateLogDisplay()
  }
}

// 創建全局實例
window.debugLogger = new DebugLogger()

// 添加日誌顯示的CSS
const style = document.createElement('style')
style.textContent = `
#debug-log-container {
  position: fixed;
  bottom: 10px;
  right: 10px;
  width: 400px;
  height: 300px;
  background: rgba(0, 0, 0, 0.9);
  color: white;
  border-radius: 8px;
  z-index: 10000;
  font-family: monospace;
  font-size: 12px;
}

#debug-log-header {
  background: #333;
  padding: 8px;
  border-radius: 8px 8px 0 0;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

#debug-log-display {
  height: 250px;
  overflow-y: auto;
  padding: 8px;
}

.log-entry {
  margin-bottom: 4px;
  padding: 2px 4px;
  border-radius: 3px;
}

.log-entry.log-info { background: rgba(0, 123, 255, 0.2); }
.log-entry.log-warn { background: rgba(255, 193, 7, 0.2); }
.log-entry.log-error { background: rgba(220, 53, 69, 0.2); }
.log-entry.log-debug { background: rgba(108, 117, 125, 0.2); }

.log-time { color: #888; }
.log-category { color: #ffc107; font-weight: bold; }
.log-message { color: white; }

.log-data {
  background: rgba(255, 255, 255, 0.1);
  padding: 4px;
  margin: 4px 0;
  border-radius: 3px;
  white-space: pre-wrap;
  font-size: 10px;
}

.debug-controls button {
  background: #007bff;
  color: white;
  border: none;
  padding: 4px 8px;
  margin-left: 4px;
  border-radius: 3px;
  font-size: 10px;
  cursor: pointer;
}

.debug-controls button:hover {
  background: #0056b3;
}
`
document.head.appendChild(style)

// 在頁面載入後添加日誌顯示區域
// document.addEventListener('DOMContentLoaded', () => {
//   const container = document.createElement('div')
//   container.id = 'debug-log-container'
//   container.innerHTML = `
//     <div id="debug-log-header">
//       <span>Debug Log</span>
//       <div class="debug-controls">
//         <button onclick="window.debugLogger.clear()">Clear</button>
//         <button onclick="window.debugLogger.exportLogs()">Export</button>
//         <button onclick="document.getElementById('debug-log-container').style.display='none'">Hide</button>
//       </div>
//     </div>
//     <div id="debug-log-display"></div>
//   `
//   document.body.appendChild(container)
// })

console.log('Debug Logger initialized')