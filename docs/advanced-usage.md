# Advanced Usage

## Multi-Device Setup with Remote Voice Synthesis

If you work on multiple devices (e.g., Mac + Jetson terminals), you can run a single voice synthesis engine on your main machine and forward the connection to other devices.

**Architecture:**
```
┌─────────────────┐     ssh -R 10101:localhost:10101
│   Mac (Server)  │◄────────────────────────────────┐
│  AivisSpeech    │                                 │
│  (port 10101)   │                                 │
└─────────────────┘                                 │
                                                    │
┌─────────────────┐  ┌─────────────────┐  ┌────────┴────────┐
│   Jetson #1     │  │   Jetson #2     │  │   Jetson #3     │
│  Speaker: A     │  │  Speaker: B     │  │  Speaker: C     │
│  (project-foo)  │  │  (project-bar)  │  │  (project-baz)  │
└─────────────────┘  └─────────────────┘  └─────────────────┘
```

**Setup:**

1. **On the server (Mac):** Start AivisSpeech or VOICEVOX

2. **On each client (Jetson):** Connect with port forwarding
   ```bash
   ssh -R 10101:localhost:10101 user@server
   ```

3. **Configure different speaker IDs per device:**
   ```json
   // Jetson #1: .claude/config.json
   {
     "default_provider": "aivisspeech",
     "providers": {
       "aivisspeech": { "speaker": 888753760 }
     }
   }

   // Jetson #2: .claude/config.json
   {
     "default_provider": "aivisspeech",
     "providers": {
       "aivisspeech": { "speaker": 1234567890 }
     }
   }
   ```

Now each device produces a distinct voice, making it easy to identify which session is speaking.
