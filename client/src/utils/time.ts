export const DAYS_OF_WEEK = [
    "monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday"
]

export const TIME_INTERVAL = 30

export const toTimeInputValue = (time: string): string => {
    if (!time) return "";
    const parts = time.split(":");
    if (parts.length < 2) return "";
    return `${parts[0].padStart(2, "0")}:${parts[1].padStart(2, "0")}`;
};

const generateTimeArray = (minInterval: number): string[] => {
    const array:string[] = []
    for (let i = 0; i <24 ; i++){
        const hour = i.toString().padStart(2, "0")
        for (let j = 0; j < (60/minInterval); j++){
            const minutes = (minInterval*j).toString().padStart(2, "0")
            array.push(hour+":"+minutes)
        }
    }
    return array;
}

const TIME_ARRAY = generateTimeArray(TIME_INTERVAL)

export const startTimeSlice = (endTime: string): string[]=> {
    const index = TIME_ARRAY.indexOf(endTime)
    if (index === -1)
        return TIME_ARRAY
    return TIME_ARRAY.slice(0, index)
}

export const endTimeSlice = (startTime: string): string[]=> {
    const index = TIME_ARRAY.indexOf(startTime)
    if (index === -1)
        return TIME_ARRAY
    return TIME_ARRAY.slice(index+1)
}