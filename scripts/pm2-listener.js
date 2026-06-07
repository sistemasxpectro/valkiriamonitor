const pm2 = require('pm2');

pm2.connect(function(err) {
    if (err) {
        console.error(err);
        process.exit(2);
    }

    pm2.launchBus(function(err, bus) {
        console.log('[PM2 Listener] Escuchando eventos críticos para Valkiria Monitor...');

        // Escucha excepciones no capturadas (Crashes de Node)
        bus.on('process:exception', function(data) {
            const appName = data.process.name;
            // Ignorar errores del propio listener para evitar bucles infinitos
            if (appName === 'pm2-listener') return; 

            const errorMsg = data.data.message || 'Excepción desconocida';
            sendAlert(appName, 'Exception (Crash)', errorMsg);
        });

        // Escucha cuando una app entra en bucle de reinicios (Restart loop)
        bus.on('process:event', function(data) {
            if (data.event === 'restart overlimit') {
                sendAlert(data.process.name, 'Restart Loop', 'La aplicación está crasheando repetidamente y PM2 dejó de reiniciarla.');
            }
        });
    });
});

function sendAlert(app, event, errorMsg) {
    const internalToken = process.env.INTERNAL_API_TOKEN || '';

    // La API fetch está disponible nativamente en Node 18+
    fetch('http://127.0.0.1:8451/pm2-alert', {
        method: 'POST',
        headers: { 
            'Content-Type': 'application/json',
            'X-Valkiria-Token': internalToken
        },
        body: JSON.stringify({
            app: app,
            event: event,
            error: errorMsg.substring(0, 200) // Truncamos por si el stacktrace es enorme
        })
    }).catch(err => {
        console.error('[PM2 Listener] Error conectando con Valkiria:', err.message);
    });
}
