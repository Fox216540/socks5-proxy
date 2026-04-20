package com.example.mobileproxy.service

import android.content.Context
import java.io.File

object LogStore {
    private const val MAX_LINES = 600
    private const val FILE_NAME = "proxy_logs.txt"

    @Synchronized
    fun add(context: Context, line: String) {
        if (line.isBlank()) {
            return
        }

        val lines = readLines(context).toMutableList()
        lines.add(line)
        while (lines.size > MAX_LINES) {
            lines.removeAt(0)
        }
        writeLines(context, lines)
    }

    @Synchronized
    fun snapshot(context: Context): List<String> = readLines(context)

    @Synchronized
    fun clear(context: Context) {
        val file = logsFile(context)
        if (file.exists()) {
            file.delete()
        }
    }

    private fun readLines(context: Context): List<String> {
        val file = logsFile(context)
        if (!file.exists()) {
            return emptyList()
        }
        return file.readLines(Charsets.UTF_8).filter { it.isNotBlank() }
    }

    private fun writeLines(context: Context, lines: List<String>) {
        val file = logsFile(context)
        file.writeText(lines.joinToString(separator = "\n", postfix = "\n"), Charsets.UTF_8)
    }

    private fun logsFile(context: Context): File {
        return File(context.filesDir, FILE_NAME)
    }
}
