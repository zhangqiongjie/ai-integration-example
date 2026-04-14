// AI Voice Chat — WebSocket Client
(function () {
    'use strict';

    // DOM refs
    const messagesEl = document.getElementById('messages');
    const input = document.getElementById('message-input');
    const sendBtn = document.getElementById('send-btn');
    const micBtn = document.getElementById('mic-btn');
    const providerSel = document.getElementById('provider');
    const voiceToggle = document.getElementById('voice-toggle');
    const voiceIcon = document.getElementById('voice-icon');
    const statusEl = document.getElementById('status');
    const recordingIndicator = document.getElementById('recording-indicator');

    // State
    let ws = null;
    let conversationId = null;
    let voiceEnabled = true;
    let currentAssistantEl = null;
    let currentAssistantText = '';

    // Audio recording state
    let mediaRecorder = null;
    let audioStream = null;
    let recording = false;
    let silenceTimer = null;
    let audioContext = null;
    let analyser = null;

    // Audio playback state
    let audioChunks = [];
    let currentAudio = null;

    // ── WebSocket ──

    function connect() {
        const proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
        let url = proto + '//' + location.host + '/ws';
        if (conversationId) url += '?conversation_id=' + conversationId;

        setStatus('connecting');
        ws = new WebSocket(url);
        ws.binaryType = 'arraybuffer';

        ws.onopen = function () { setStatus('connected'); };

        ws.onclose = function () {
            setStatus('disconnected');
            setTimeout(connect, 3000);
        };

        ws.onerror = function () { setStatus('disconnected'); };

        ws.onmessage = function (evt) {
            if (evt.data instanceof ArrayBuffer) {
                handleBinaryMessage(evt.data);
                return;
            }
            var msg;
            try { msg = JSON.parse(evt.data); } catch (e) { return; }
            handleServerMessage(msg);
        };
    }

    function handleServerMessage(msg) {
        switch (msg.type) {
            case 'init':
                conversationId = msg.conversation_id;
                voiceEnabled = msg.voice_enabled;
                updateVoiceIcon();
                if (msg.provider) providerSel.value = msg.provider;
                break;

            case 'transcription':
                addMessage('user', msg.content, 'voice');
                break;

            case 'response_chunk':
                if (!currentAssistantEl) {
                    currentAssistantEl = createMessageEl('assistant');
                    currentAssistantText = '';
                    messagesEl.appendChild(currentAssistantEl);
                }
                currentAssistantText += msg.content;
                currentAssistantEl.querySelector('.content').textContent = currentAssistantText;
                scrollToBottom();
                break;

            case 'response_done':
                if (currentAssistantEl) {
                    currentAssistantText = msg.full_content || currentAssistantText;
                    currentAssistantEl.querySelector('.content').textContent = currentAssistantText;
                }
                currentAssistantEl = null;
                currentAssistantText = '';
                audioChunks = [];
                scrollToBottom();
                break;

            case 'tts_done':
                playAudioChunks();
                break;

            case 'provider_changed':
                providerSel.value = msg.provider;
                break;

            case 'error':
                addSystemMessage('Error: ' + msg.error);
                break;
        }
    }

    function handleBinaryMessage(data) {
        audioChunks.push(data);
    }

    function sendText(text) {
        if (!ws || ws.readyState !== WebSocket.OPEN) return;
        if (!text.trim()) {
            addSystemMessage('Please enter a message.');
            return;
        }
        ws.send(JSON.stringify({ type: 'text', content: text }));
        addMessage('user', text, 'text');
        input.value = '';
    }

    // ── Messages UI ──

    function createMessageEl(role) {
        var el = document.createElement('div');
        el.className = 'message ' + role;
        var content = document.createElement('div');
        content.className = 'content';
        el.appendChild(content);
        return el;
    }

    function addMessage(role, text, inputMode) {
        var el = createMessageEl(role);
        el.querySelector('.content').textContent = text;
        if (inputMode === 'voice') {
            var badge = document.createElement('span');
            badge.className = 'input-badge';
            badge.textContent = 'voice';
            el.appendChild(badge);
        }
        messagesEl.appendChild(el);
        scrollToBottom();
    }

    function addSystemMessage(text) {
        var el = document.createElement('div');
        el.className = 'message assistant';
        el.style.opacity = '0.7';
        el.style.fontStyle = 'italic';
        var content = document.createElement('div');
        content.className = 'content';
        content.textContent = text;
        el.appendChild(content);
        messagesEl.appendChild(el);
        scrollToBottom();
    }

    function scrollToBottom() {
        messagesEl.scrollTop = messagesEl.scrollHeight;
    }

    function setStatus(state) {
        statusEl.className = 'status ' + state;
        statusEl.textContent = state.charAt(0).toUpperCase() + state.slice(1);
    }

    // ── Voice Recording ──

    async function startRecording() {
        try {
            audioStream = await navigator.mediaDevices.getUserMedia({ audio: true });
        } catch (err) {
            addSystemMessage('Microphone access denied. Please allow microphone access or use text input.');
            return;
        }

        recording = true;
        micBtn.classList.add('recording');
        recordingIndicator.classList.remove('hidden');

        // Stop any playing audio.
        stopAudioPlayback();

        ws.send(JSON.stringify({ type: 'voice_start', format: 'webm' }));

        mediaRecorder = new MediaRecorder(audioStream, { mimeType: 'audio/webm' });
        mediaRecorder.ondataavailable = function (e) {
            if (e.data.size > 0 && ws && ws.readyState === WebSocket.OPEN) {
                e.data.arrayBuffer().then(function (buf) {
                    ws.send(buf);
                });
            }
        };
        mediaRecorder.start(250); // send chunks every 250ms

        // Silence detection.
        startSilenceDetection(audioStream);
    }

    function stopRecording() {
        recording = false;
        micBtn.classList.remove('recording');
        recordingIndicator.classList.add('hidden');

        if (silenceTimer) { clearTimeout(silenceTimer); silenceTimer = null; }
        if (analyser) { analyser = null; }
        if (audioContext) { audioContext.close().catch(function () {}); audioContext = null; }

        if (mediaRecorder && mediaRecorder.state !== 'inactive') {
            mediaRecorder.stop();
        }
        if (audioStream) {
            audioStream.getTracks().forEach(function (t) { t.stop(); });
            audioStream = null;
        }

        if (ws && ws.readyState === WebSocket.OPEN) {
            ws.send(JSON.stringify({ type: 'voice_end' }));
        }
    }

    function startSilenceDetection(stream) {
        audioContext = new (window.AudioContext || window.webkitAudioContext)();
        var source = audioContext.createMediaStreamSource(stream);
        analyser = audioContext.createAnalyser();
        analyser.fftSize = 512;
        source.connect(analyser);

        var dataArray = new Uint8Array(analyser.frequencyBinCount);
        var silenceStart = null;
        var SILENCE_THRESHOLD = 10;
        var SILENCE_DURATION = 1500; // ms

        function check() {
            if (!recording || !analyser) return;
            analyser.getByteFrequencyData(dataArray);
            var avg = 0;
            for (var i = 0; i < dataArray.length; i++) avg += dataArray[i];
            avg /= dataArray.length;

            if (avg < SILENCE_THRESHOLD) {
                if (!silenceStart) silenceStart = Date.now();
                if (Date.now() - silenceStart > SILENCE_DURATION) {
                    stopRecording();
                    return;
                }
            } else {
                silenceStart = null;
            }
            requestAnimationFrame(check);
        }
        requestAnimationFrame(check);
    }

    // ── Audio Playback ──

    function playAudioChunks() {
        if (audioChunks.length === 0) return;
        var blob = new Blob(audioChunks, { type: 'audio/mpeg' });
        var url = URL.createObjectURL(blob);
        currentAudio = new Audio(url);
        currentAudio.onended = function () {
            URL.revokeObjectURL(url);
            currentAudio = null;
        };
        currentAudio.play().catch(function () {});
        audioChunks = [];
    }

    function stopAudioPlayback() {
        if (currentAudio) {
            currentAudio.pause();
            currentAudio = null;
        }
        audioChunks = [];
    }

    // ── Voice Toggle ──

    function updateVoiceIcon() {
        voiceIcon.innerHTML = voiceEnabled ? '&#128264;' : '&#128263;';
    }

    // ── Provider Switch ──

    providerSel.addEventListener('change', function () {
        if (ws && ws.readyState === WebSocket.OPEN) {
            ws.send(JSON.stringify({ type: 'provider_switch', provider: providerSel.value }));
        }
    });

    // ── Event Listeners ──

    sendBtn.addEventListener('click', function () { sendText(input.value); });

    input.addEventListener('keydown', function (e) {
        if (e.key === 'Enter' && !e.shiftKey) {
            e.preventDefault();
            sendText(input.value);
        }
    });

    voiceToggle.addEventListener('click', function () {
        voiceEnabled = !voiceEnabled;
        updateVoiceIcon();
        if (ws && ws.readyState === WebSocket.OPEN) {
            ws.send(JSON.stringify({ type: 'voice_toggle', enabled: voiceEnabled }));
        }
    });

    micBtn.addEventListener('mousedown', function (e) {
        e.preventDefault();
        if (!recording) startRecording();
    });

    micBtn.addEventListener('mouseup', function () {
        if (recording) stopRecording();
    });

    micBtn.addEventListener('mouseleave', function () {
        if (recording) stopRecording();
    });

    // Touch support for mobile.
    micBtn.addEventListener('touchstart', function (e) {
        e.preventDefault();
        if (!recording) startRecording();
    });

    micBtn.addEventListener('touchend', function (e) {
        e.preventDefault();
        if (recording) stopRecording();
    });

    // ── Init ──
    connect();
})();
