import logging
import os
import json
import pika
from dotenv import load_dotenv
from google import genai
from google.genai import types
import random 
import time

logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(message)s')
LOGGER = logging.getLogger(__name__)

load_dotenv()

class AsyncAIWorker:
    def __init__(self):
        self._connection = None
        self._channel = None
        self._closing = False
        self._url = os.getenv("RABBITMQ_URL")
        
        # Google AI Client (v1.0)
        api_key = os.getenv("GOOGLE_API_KEY")
        if not api_key:
            raise ValueError("Failed to find GOOGLE_API_KEY!")
        self.ai_client = genai.Client(api_key=api_key)

    def run(self):
        LOGGER.info('Starting RabbitMQ connection ...')
        self._connection = pika.SelectConnection(
            pika.URLParameters(self._url),
            on_open_callback=self.on_connection_open,
            on_open_error_callback=self.on_connection_error,
            on_close_callback=self.on_connection_closed)
        self._connection.ioloop.start()

    def on_connection_open(self, _unused_connection):
        LOGGER.info('Opened connection. Wants channel ...')
        self._connection.channel(on_open_callback=self.on_channel_open)

    def on_channel_open(self, channel):
        LOGGER.info('Opened channel. Initiliazes queues...')
        self._channel = channel
        self._channel.add_on_close_callback(self.on_channel_closed)
        
        
        self._channel.queue_declare(
            queue='logs_queue', 
            durable=True, 
            callback=self.on_logs_queue_declared 
        )

    def on_logs_queue_declared(self, _unused_frame):
        LOGGER.info('logs_queue is ready.')
        
        
        self._channel.queue_declare(
            queue='incidents_queue', 
            durable=True, 
            callback=self.on_incidents_queue_declared 
        )

    
    def on_incidents_queue_declared(self, _unused_frame):
        LOGGER.info('incidents_queue is ready.')
        self._channel.basic_qos(prefetch_count=1, callback=self.on_basic_qos_ok)

    def on_basic_qos_ok(self, _unused_frame):
        LOGGER.info('Starting is consumpting...')
        self._channel.basic_consume('logs_queue', self.on_message)

    def on_message(self, _unused_channel, basic_deliver, properties, body):
        try:
            log_entry = json.loads(body)
            LOGGER.info(f"📥 [INPUT] {log_entry.get('service')}")
            
            
            analysis = self.analyze_with_ai(log_entry)
            
            forward_data = {
                "original_log": log_entry.get("message"), 
                "analysis": f"Risk: {analysis.get('risk')} | {analysis.get('explanation')}", 
                "solution": analysis.get("action") 
            }
            
            self.publish_incident(forward_data)
            
            self._channel.basic_ack(basic_deliver.delivery_tag)
            
        except Exception as e:
            LOGGER.error(f"Hata: {e}")
            self._channel.basic_ack(basic_deliver.delivery_tag)

    def publish_incident(self, incident_data):
        try:
            json_body = json.dumps(incident_data)
            
            self._channel.basic_publish(
                exchange='',
                routing_key='incidents_queue', # target queue
                body=json_body,
                properties=pika.BasicProperties(
                    delivery_mode=2, # perminant message
                    content_type='application/json'
                )
            )
            LOGGER.info(f"📤 [SUCCESS] -> incidents_queue | Solution: {incident_data.get('solution')}")
        except Exception as e:
            LOGGER.error(f"Error: {e}")

    def analyze_with_ai(self, log_data):

        """
        🛑 KOTA DOLDUĞUNDA KULLANILACAK MOD (MOCK MODE)
        """
        LOGGER.info("⚠️ KOTA DOLU: Mock (Taklit) veri üretiliyor...")
        
        time.sleep(1) 
        
        scenarios = [
            {"risk": "HIGH", "action": "Restart Service", "explanation": "CPU Overheat detected (Simulated)"},
            {"risk": "CRITICAL", "action": "Call Admin", "explanation": "Database Connection Lost (Simulated)"},
            {"risk": "LOW", "action": "Log Info", "explanation": "Routine Check (Simulated)"},
            {"risk": "MEDIUM", "action": "Clear Cache", "explanation": "Memory Warning (Simulated)"}
        ]
        
        return random.choice(scenarios)
        """
        try:
            prompt = f
            Analyze this log JSON as a System Admin. Return JSON only.
            Log: {json.dumps(log_data)}
            Format: {{"risk": "LOW/HIGH", "action": "Suggested Action", "explanation": "Short info"}}
            
            response = self.ai_client.models.generate_content(
                model='gemini-1.5-flash',
                contents=prompt,
                config=types.GenerateContentConfig(temperature=0.1)
            )
            return json.loads(response.text.replace("```json", "").replace("```", "").strip())
        except:
            return {"risk": "UNKNOWN", "action": "Manual Check", "explanation": "AI Fail"}"""

    
    def on_connection_error(self, _unused_connection, err):
        LOGGER.error(f"Connection Error: {err}")
        self.stop()

    def on_channel_closed(self, channel, reason):
        LOGGER.warning(f"Channel is closed: {reason}")
        self._connection.close()

    def on_connection_closed(self, _unused_connection, reason):
        LOGGER.warning(f"Connection is closed: {reason}")
        self._connection.ioloop.stop()

    def stop(self):
        LOGGER.info('Closing...')
        self._connection.close()

if __name__ == '__main__':
    worker = AsyncAIWorker()
    try:
        worker.run()
    except KeyboardInterrupt:
        worker.stop()