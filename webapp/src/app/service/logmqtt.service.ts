import { Injectable } from '@angular/core';
import { IMqttMessage, MqttService } from 'ngx-mqtt';
import { Observable } from 'rxjs';

@Injectable()
export class LogMqttService {

    constructor(private _mqttService: MqttService) {

    }
    topic(val: string): Observable<IMqttMessage> {
        return this._mqttService.observe(val);

    }
}
