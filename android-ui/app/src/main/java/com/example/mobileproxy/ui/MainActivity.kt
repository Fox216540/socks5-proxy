package com.example.mobileproxy.ui

import android.content.Intent
import android.content.IntentFilter
import android.content.res.ColorStateList
import android.os.Build
import android.os.Bundle
import android.text.format.DateFormat
import android.widget.Button
import android.widget.EditText
import android.widget.ScrollView
import android.widget.Switch
import android.widget.TextView
import android.widget.Toast
import androidx.appcompat.widget.SwitchCompat
import androidx.appcompat.app.AppCompatActivity
import com.example.mobileproxy.R
import com.example.mobileproxy.service.LogStore
import com.example.mobileproxy.service.ProxyService
import java.util.Date

class MainActivity : AppCompatActivity() {
    private lateinit var logsText: TextView
    private lateinit var logsScroll: ScrollView
    private lateinit var runSwitch: SwitchCompat
    private var suppressSwitchCallback = false

    private val logReceiver = object : android.content.BroadcastReceiver() {
        override fun onReceive(context: android.content.Context?, intent: Intent?) {
            if (intent?.action != ProxyService.ACTION_LOG) {
                return
            }
            val line = intent.getStringExtra(ProxyService.EXTRA_LOG_LINE) ?: return
            appendLog(line, persist = false)
        }
    }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContentView(R.layout.activity_main)

        val ipInput = findViewById<EditText>(R.id.ipInput)
        val portInput = findViewById<EditText>(R.id.portInput)
        val tlsSwitch = findViewById<Switch>(R.id.tlsSwitch)
        runSwitch = findViewById(R.id.runSwitch)
        val clearLogsBtn = findViewById<Button>(R.id.clearLogsBtn)
        logsText = findViewById(R.id.logsText)
        logsScroll = findViewById(R.id.logsScroll)
        hydrateLogs()
        applyRunSwitchUi(runSwitch, false)

        runSwitch.setOnCheckedChangeListener { _, isChecked ->
            if (suppressSwitchCallback) {
                return@setOnCheckedChangeListener
            }
            applyRunSwitchUi(runSwitch, isChecked)

            if (isChecked) {
                val ip = ipInput.text.toString().trim()
                val portText = portInput.text.toString().trim()

                if (ip.isEmpty()) {
                    Toast.makeText(this, "IP is required", Toast.LENGTH_SHORT).show()
                    setRunSwitchChecked(runSwitch, false)
                    return@setOnCheckedChangeListener
                }

                val port = portText.toIntOrNull()
                if (port == null || port !in 1..65535) {
                    Toast.makeText(this, "Port must be 1..65535", Toast.LENGTH_SHORT).show()
                    setRunSwitchChecked(runSwitch, false)
                    return@setOnCheckedChangeListener
                }

                val address = "$ip:$port"
                val useTls = tlsSwitch.isChecked

                val intent = Intent(this, ProxyService::class.java).apply {
                    action = ProxyService.ACTION_START
                    putExtra(ProxyService.EXTRA_ADDR, address)
                    putExtra(ProxyService.EXTRA_TLS, useTls)
                }

                if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
                    startForegroundService(intent)
                } else {
                    startService(intent)
                }

                appendLog("Start requested: $address (tls=$useTls)", persist = true)
                Toast.makeText(this, "Started: $address", Toast.LENGTH_SHORT).show()
            } else {
                val intent = Intent(this, ProxyService::class.java).apply {
                    action = ProxyService.ACTION_STOP
                }

                if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
                    startForegroundService(intent)
                } else {
                    startService(intent)
                }

                appendLog("Stop requested", persist = true)
                Toast.makeText(this, "Stopped", Toast.LENGTH_SHORT).show()
            }
        }

        clearLogsBtn.setOnClickListener {
            LogStore.clear(this)
            logsText.text = ""
            Toast.makeText(this, "Logs cleared", Toast.LENGTH_SHORT).show()
        }
    }

    override fun onStart() {
        super.onStart()
        val filter = IntentFilter(ProxyService.ACTION_LOG)
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
            registerReceiver(logReceiver, filter, RECEIVER_NOT_EXPORTED)
        } else {
            @Suppress("DEPRECATION")
            registerReceiver(logReceiver, filter)
        }
        syncRunSwitchState()
    }

    override fun onStop() {
        try {
            unregisterReceiver(logReceiver)
        } catch (_: IllegalArgumentException) {
            // Receiver may already be unregistered on rapid lifecycle changes.
        }
        super.onStop()
    }

    private fun appendLog(message: String, persist: Boolean) {
        val line = if (looksTimestamped(message)) {
            message
        } else {
            val time = DateFormat.format("HH:mm:ss", Date()).toString()
            "[$time] $message"
        }
        if (persist) {
            LogStore.add(this, line)
        }
        if (logsText.text.isNullOrEmpty()) {
            logsText.text = line
        } else {
            logsText.append("\n$line")
        }
        logsScroll.post { logsScroll.fullScroll(ScrollView.FOCUS_DOWN) }
    }

    private fun looksTimestamped(message: String): Boolean {
        return message.length >= 11 &&
            message.startsWith("[") &&
            message[3] == ':' &&
            message[6] == ':' &&
            message[9] == ']' &&
            message[10] == ' '
    }

    private fun hydrateLogs() {
        val lines = LogStore.snapshot(this)
        if (lines.isEmpty()) {
            return
        }
        logsText.text = lines.joinToString("\n")
        logsScroll.post { logsScroll.fullScroll(ScrollView.FOCUS_DOWN) }
    }

    private fun setRunSwitchChecked(runSwitch: SwitchCompat, checked: Boolean) {
        suppressSwitchCallback = true
        runSwitch.isChecked = checked
        suppressSwitchCallback = false
        applyRunSwitchUi(runSwitch, checked)
    }

    private fun applyRunSwitchUi(runSwitch: SwitchCompat, isOn: Boolean) {
        val accent = if (isOn) 0xFF1B8F3A.toInt() else 0xFFB3261E.toInt()
        val track = if (isOn) 0x661B8F3A.toInt() else 0x66B3261E.toInt()

        runSwitch.text = if (isOn) "Proxy ON" else "Proxy OFF"
        runSwitch.setTextColor(accent)
        runSwitch.thumbTintList = ColorStateList.valueOf(accent)
        runSwitch.trackTintList = ColorStateList.valueOf(track)
    }

    private fun syncRunSwitchState() {
        val running = ProxyService.isRunning(this)
        setRunSwitchChecked(runSwitch, running)
    }
}
