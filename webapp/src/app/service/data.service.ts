import { HttpClient } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { EMPTY, Observable, of } from 'rxjs';
import { HostStates } from '../models/hostStates';
import { LogEntry } from '../models/logEntry';

@Injectable({
  providedIn: 'root'
})
export class DataService {
  private readonly baseUrl = 'rest/log/';
  constructor(private http: HttpClient) { }

  findLogs(host = 'ALL', target = 'ALL', sortOrder = 'DESC', level = 'ALL', limit = 10, offset = 0): Observable<LogEntry[]> {
    if (host === '') {
      host = ' ';
    }
    return this.http.get<LogEntry[]>(`${this.baseUrl}${host}/${target}/${level}/${limit}/${offset}/${sortOrder}`);
  }
  getLevels(): Observable<string[]> {
    return this.http.get<string[]>(this.baseUrl + 'levels');
  }
  getHosts(): Observable<Map<string, HostStates>> {
    return this.http.get<Map<string, HostStates>>(this.baseUrl + 'hosts');
  }
  getSize(host: string, level: string, target: string) {
    if (host === '') {
      host = ' ';
    }
    return this.http.get<number>(`${this.baseUrl}${host}/${target}/${level}`);
  }
  getTargets(host: string): Observable<string[]> {
    return this.http.get<string[]>(this.baseUrl + 'targets/' + `${host}`);
  }

  setLogLevel(host: string, target: string, level: string) {
    if (host === '') {
      host = ' ';
    }
    return this.http.put<string>(`${this.baseUrl}${host}/${target}`, level);
  }
  deleteLogs(host: string, target: string) {
    if (host === '') {
      host = ' ';
    }
    return this.http.delete(`${this.baseUrl}${host}/${target}`);
  }
}
