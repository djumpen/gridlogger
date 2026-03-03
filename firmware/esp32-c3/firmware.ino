#include <HTTPClient.h>
#include <WiFi.h>
#include <WiFiClientSecure.h>

#include "generated_config.h"

const unsigned long kPingIntervalMs = 30000;
const unsigned long kWifiRetryIntervalMs = 10000;
const unsigned long kLedBlinkIntervalMs = 450;

// ESP32-C3 Super Mini boards usually have built-in blue LED on GPIO8 (active-low).
const int kLedPin = 8;
const bool kLedActiveLow = true;

unsigned long gLastPingAtMs = 0;
unsigned long gLastWifiAttemptAtMs = 0;
unsigned long gLastLedBlinkAtMs = 0;
bool gWifiConnected = false;
bool gLedBlinkState = false;

void setLed(bool on) {
  digitalWrite(kLedPin, (kLedActiveLow ? !on : on) ? HIGH : LOW);
}

void pulseLed(unsigned long durationMs) {
  setLed(true);
  delay(durationMs);
  setLed(false);
}

const char* wifiStatusToText(wl_status_t st) {
  switch (st) {
    case WL_NO_SHIELD: return "WL_NO_SHIELD";
    case WL_IDLE_STATUS: return "WL_IDLE_STATUS";
    case WL_NO_SSID_AVAIL: return "WL_NO_SSID_AVAIL";
    case WL_SCAN_COMPLETED: return "WL_SCAN_COMPLETED";
    case WL_CONNECTED: return "WL_CONNECTED";
    case WL_CONNECT_FAILED: return "WL_CONNECT_FAILED";
    case WL_CONNECTION_LOST: return "WL_CONNECTION_LOST";
    case WL_DISCONNECTED: return "WL_DISCONNECTED";
    default: return "WL_UNKNOWN";
  }
}

bool sendPing() {
  pulseLed(70);
  WiFiClientSecure client;
  client.setInsecure();

  HTTPClient http;
  if (!http.begin(client, GRID_PING_URL)) {
    Serial.println("Не вдалося ініціалізувати HTTP клієнт");
    return false;
  }

  http.setTimeout(15000);
  http.addHeader("X-Project-Secret", GRID_PROJECT_SECRET);

  int httpCode = http.POST("");
  if (httpCode >= 200 && httpCode < 300) {
    Serial.printf("Ping успішний: %d\n", httpCode);
    http.end();
    return true;
  }

  if (httpCode > 0) {
    Serial.printf("Ping HTTP помилка: %d\n", httpCode);
  } else {
    Serial.printf("Ping помилка: %s\n", http.errorToString(httpCode).c_str());
  }

  http.end();
  return false;
}

void ensureWifiConnected() {
  wl_status_t status = WiFi.status();
  if (status == WL_CONNECTED) {
    if (!gWifiConnected) {
      gWifiConnected = true;
      Serial.print("WiFi підключено. IP: ");
      Serial.println(WiFi.localIP());
      setLed(true);
    }
    return;
  }

  if (gWifiConnected) {
    gWifiConnected = false;
    Serial.println("WiFi втрачено");
    setLed(false);
  }

  unsigned long now = millis();
  if (now - gLastLedBlinkAtMs >= kLedBlinkIntervalMs || gLastLedBlinkAtMs == 0) {
    gLedBlinkState = !gLedBlinkState;
    setLed(gLedBlinkState);
    gLastLedBlinkAtMs = now;
  }

  if (now - gLastWifiAttemptAtMs >= kWifiRetryIntervalMs || gLastWifiAttemptAtMs == 0) {
    Serial.printf("Повторне підключення до WiFi... status=%s (%d)\n", wifiStatusToText(status), static_cast<int>(status));
    WiFi.begin(GRID_WIFI_SSID, GRID_WIFI_PASSWORD);
    gLastWifiAttemptAtMs = now;
  }
}

void setup() {
  Serial.begin(115200);
  delay(1500);
  pinMode(kLedPin, OUTPUT);
  setLed(false);
  pulseLed(120);
  delay(80);
  pulseLed(120);

  WiFi.mode(WIFI_STA);
  WiFi.setSleep(false);

  Serial.println("GridLogger ESP32-C3 firmware startup");
  Serial.printf("SSID: %s\n", GRID_WIFI_SSID);
  Serial.printf("Ping URL: %s\n", GRID_PING_URL);

  WiFi.begin(GRID_WIFI_SSID, GRID_WIFI_PASSWORD);
}

void loop() {
  ensureWifiConnected();
  if (WiFi.status() != WL_CONNECTED) {
    delay(150);
    return;
  }

  setLed(true);
  unsigned long now = millis();
  if (gLastPingAtMs == 0 || now - gLastPingAtMs >= kPingIntervalMs) {
    if (sendPing()) {
      gLastPingAtMs = now;
      setLed(true);
    } else {
      // Retry faster after transient error.
      gLastPingAtMs = now - (kPingIntervalMs - 5000);
      setLed(false);
    }
  }

  delay(20);
}
