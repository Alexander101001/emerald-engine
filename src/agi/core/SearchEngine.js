import axios from 'axios';

export class SearchEngine {
    constructor(apiKey) {
        this.apiKey = apiKey;
    }

    async search(query) {
        if (!this.apiKey || this.apiKey.startsWith('your_')) {
            return [];
        }
        try {
            console.log(`[SEARCH] Querying internet: ${query}`);
            const response = await axios.get('https://google.serper.dev/search', {
                params: { q: query },
                headers: { 'X-API-KEY': this.apiKey }
            });
            return response.data.organic.slice(0, 3);
        } catch (error) {
            return [];
        }
    }
}
