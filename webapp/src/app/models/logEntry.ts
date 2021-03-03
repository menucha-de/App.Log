export interface LogEntry {
    id?: string;
    time?: number;
    host?: string;
    targetName?: string;
    sourceMethod?: string;
    level?: string;
    message?: string;
    thrown?: string;
}
