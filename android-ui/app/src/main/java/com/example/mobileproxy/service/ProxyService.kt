package com.example.mobileproxy.service

import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.Service
import android.content.Context
import android.content.Intent
import android.os.Build
import android.os.IBinder
import android.text.format.DateFormat
import androidx.core.app.NotificationCompat
import client.client.Client
import client.client.Logger
import java.util.Date

class ProxyService : Service() {
    private val goLogger = object : Logger {
        override fun onLog(line: String) {
            emitLog("[go] $line")
        }
    }

    override fun onCreate() {
        super.onCreate()
        Client.setLogger(goLogger)
        emitLog("Go logger attached")
    }

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        return try {
            when (intent?.action) {
                ACTION_STOP -> {
                    setRunning(this, false)
                    Client.stop()
                    emitLog("Client stopped")
                    stopForegroundCompat()
                    stopSelf()
                    START_NOT_STICKY
                }
                ACTION_START, null -> {
                    val addr = intent?.getStringExtra(EXTRA_ADDR) ?: return START_NOT_STICKY
                    val useTls = intent.getBooleanExtra(EXTRA_TLS, false)

                    startForeground(NOTIFICATION_ID, createNotification(addr, useTls))
                    Client.startWithTLS(addr, useTls)
                    setRunning(this, true)
                    emitLog("Client started: $addr (tls=$useTls)")

                    START_STICKY
                }
                else -> START_NOT_STICKY
            }
        } catch (t: Throwable) {
            setRunning(this, false)
            emitLog("Start failed: ${t.javaClass.simpleName}: ${t.message}")
            stopSelf()
            START_NOT_STICKY
        }
    }

    override fun onDestroy() {
        setRunning(this, false)
        Client.clearLogger()
        Client.stop()
        emitLog("Service destroyed")
        super.onDestroy()
    }

    override fun onBind(intent: Intent?): IBinder? = null

    private fun stopForegroundCompat() {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.N) {
            stopForeground(STOP_FOREGROUND_REMOVE)
        } else {
            @Suppress("DEPRECATION")
            stopForeground(true)
        }
    }

    private fun createNotification(addr: String, useTls: Boolean): Notification {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            val channel = NotificationChannel(
                CHANNEL_ID,
                "Proxy Service",
                NotificationManager.IMPORTANCE_LOW
            )
            val manager = getSystemService(NotificationManager::class.java)
            manager.createNotificationChannel(channel)
        }

        val mode = if (useTls) "TLS on" else "TLS off"
        return NotificationCompat.Builder(this, CHANNEL_ID)
            .setSmallIcon(android.R.drawable.stat_notify_sync)
            .setContentTitle("Reverse SOCKS is running")
            .setContentText("$addr ($mode)")
            .setOngoing(true)
            .build()
    }

    private fun emitLog(message: String) {
        val time = DateFormat.format("HH:mm:ss", Date()).toString()
        val line = "[$time] $message"
        LogStore.add(this, line)
        sendBroadcast(
            Intent(ACTION_LOG).apply {
                `package` = packageName
                putExtra(EXTRA_LOG_LINE, line)
            }
        )
    }

    companion object {
        const val ACTION_START = "com.example.mobileproxy.action.START"
        const val ACTION_STOP = "com.example.mobileproxy.action.STOP"
        const val ACTION_LOG = "com.example.mobileproxy.action.LOG"

        const val EXTRA_ADDR = "addr"
        const val EXTRA_TLS = "tls"
        const val EXTRA_LOG_LINE = "log_line"

        private const val STATE_PREFS = "proxy_state"
        private const val KEY_RUNNING = "running"
        private const val CHANNEL_ID = "proxy_service_channel"
        private const val NOTIFICATION_ID = 1

        fun isRunning(context: Context): Boolean {
            return context.getSharedPreferences(STATE_PREFS, Context.MODE_PRIVATE)
                .getBoolean(KEY_RUNNING, false)
        }

        private fun setRunning(context: Context, running: Boolean) {
            context.getSharedPreferences(STATE_PREFS, Context.MODE_PRIVATE)
                .edit()
                .putBoolean(KEY_RUNNING, running)
                .apply()
        }
    }
}
